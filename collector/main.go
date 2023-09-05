package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kelseyhightower/envconfig"

	gpprof "github.com/google/pprof/profile"
)

type config struct {
	BindAddress        string `envconfig:"bind_address" default:"127.0.0.1:8080"`
	Directory          string `envconfig:"directory" default:"./profiles"`
	SubmitAuthKeysFile string `envconfig:"submit_auth_keys_file" default:"/secret/submit_keys"`
	MergeAuthKeysFile  string `envconfig:"merge_auth_keys_file" default:"/secret/merge_keys"`
}

var c config
var submitKeys []string
var mergeKeys []string
var index = new(atomic.Int64)
var mergeMu = new(sync.Mutex)

func main() {
	err := envconfig.Process("pgof", &c)
	if err != nil {
		log.Fatal(err)
	}

	submitKeys = readKeys(c.SubmitAuthKeysFile)
	mergeKeys = readKeys(c.MergeAuthKeysFile)

	err = os.MkdirAll(c.Directory, os.ModeDir)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(http.StatusText(http.StatusOK)))
	})
	http.HandleFunc("/v1/submit", httpSubmit)
	http.HandleFunc("/v1/merge", httpMerge)
	err = http.ListenAndServe(c.BindAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func httpSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !checkAuth(r, submitKeys) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	defer func() {
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
	}()

	// Parse the profile
	profile, err := gpprof.Parse(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check to make sure it's a CPU profile
	if profile.PeriodType.Type != "cpu" || profile.PeriodType.Unit != "nanoseconds" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(profile.SampleType) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hasSamples := false
	hasCpu := false
	for _, s := range profile.SampleType {
		if s.Type == "samples" && s.Unit == "count" && !hasSamples {
			hasSamples = true
		} else if s.Type == "cpu" && s.Unit == "nanoseconds" && !hasCpu {
			hasCpu = true
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if !hasSamples && !hasCpu {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Persist that profile
	buf := new(bytes.Buffer)
	if err = profile.Write(buf); err != nil {
		log.Println("[SUBMIT]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fname := fmt.Sprintf("%d_%d.pgo", time.Now().UnixMilli(), index.Add(1))
	if err = os.WriteFile(path.Join(c.Directory, fname), buf.Bytes(), 0664); err != nil {
		log.Println("[SUBMIT]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("[SUBMIT] Received profile was persisted as ", fname)
	w.WriteHeader(http.StatusNoContent)
}

func httpMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !checkAuth(r, mergeKeys) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	combine := r.URL.Query().Get("and_combine") == "true"

	mergeMu.Lock()
	defer mergeMu.Unlock()

	profiles := make([]*gpprof.Profile, 0)
	entries, err := os.ReadDir(c.Directory)
	if err != nil {
		log.Println("[MERGE]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	flagged := make([]string, 0)
	var f *os.File
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if f, err = os.Open(path.Join(c.Directory, e.Name())); err != nil {
			log.Println("[MERGE]", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		flagged = append(flagged, f.Name())
		if profile, err := gpprof.Parse(f); err != nil {
			log.Println("[MERGE]", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			profiles = append(profiles, profile)
		}
		if err = f.Close(); err != nil {
			log.Println("[MERGE]", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	merged, err := gpprof.Merge(profiles)
	if err != nil {
		log.Println("[MERGE]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	if err = merged.Write(buf); err != nil {
		log.Println("[MERGE]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if combine {
		fname := fmt.Sprintf("merged_%d_%d.pgo", time.Now().UnixMilli(), index.Add(1))
		if err = os.WriteFile(path.Join(c.Directory, fname), buf.Bytes(), 0664); err != nil {
			log.Println("[MERGE]", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("[MERGE] Combined to ", fname)

		for _, n := range flagged {
			if err = os.Remove(n); err != nil {
				log.Println("[MERGE]", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	log.Println("[MERGE] Writing merged profile as HTTP response")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(buf.Bytes()); err != nil {
		log.Println("[MERGE-ERR_WRITE]", err)
	}
}

func checkAuth(r *http.Request, keys []string) bool {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return false
	}
	h = h[len("Bearer "):]
	for _, k := range keys {
		if k == h {
			return true
		}
	}
	return false
}

func readKeys(p string) []string {
	b, err := os.ReadFile(p)
	if err != nil {
		log.Fatal(err)
	}
	return splitLines(string(b))
}

func splitLines(s string) []string {
	parts := strings.Split(s, "\n")
	lines := make([]string, 0)
	for _, s2 := range parts {
		t := strings.TrimSpace(s2)
		if len(t) > 0 {
			lines = append(lines, t)
		}
	}
	return lines
}
