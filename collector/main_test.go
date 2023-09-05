package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	gpprof "github.com/google/pprof/profile"
)

var sampleProfile []byte

func init() {
	b, err := os.ReadFile("./testdata/cpu.pgo")
	if err != nil {
		panic(err)
	}
	sampleProfile = b

	submitKeys = []string{"test!1234"}
	mergeKeys = []string{"test_1234"}
}

func makeConfig(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "pgo_test")
	if err != nil {
		t.Fatal(err)
	}
	c = config{
		BindAddress:        "",
		Directory:          dir,
		SubmitAuthKeysFile: "",
		MergeAuthKeysFile:  "",
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSubmitHandler(t *testing.T) {
	makeConfig(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/submit", bytes.NewReader(sampleProfile))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", submitKeys[0]))
	w := httptest.NewRecorder()
	httpSubmit(w, req)
	res := w.Result()
	defer func(rbody io.ReadCloser) {
		_ = rbody.Close()
	}(res.Body)
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status code %d but got %d", http.StatusNoContent, res.StatusCode)
	}
}

func TestMergeHandler(t *testing.T) {
	makeConfig(t)

	if err := os.WriteFile(path.Join(c.Directory, "cpu1.pgo"), sampleProfile, 0664); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path.Join(c.Directory, "cpu2.pgo"), sampleProfile, 0664); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/merge", bytes.NewReader(sampleProfile))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mergeKeys[0]))
	w := httptest.NewRecorder()
	httpMerge(w, req)
	res := w.Result()
	defer func(rbody io.ReadCloser) {
		_ = rbody.Close()
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, res.StatusCode)
	}

	profile, err := gpprof.Parse(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	ogProfile, err := gpprof.Parse(bytes.NewReader(sampleProfile))
	if err != nil {
		t.Fatal(err)
	}

	if len(profile.Sample) != len(ogProfile.Sample) {
		t.Errorf("Expected %d samples but got %d", len(ogProfile.Sample), len(profile.Sample))
	}
	if (profile.Sample[0].Value[0] / 2) != ogProfile.Sample[0].Value[0] {
		t.Error("Expected value to be double the original")
	}
}

// TODO: Should probably add edge case tests too
