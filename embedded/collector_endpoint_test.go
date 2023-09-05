package pgo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectorEndpoint_Submit(t *testing.T) {
	key := "test1234"
	receiveData := []byte("hello world")

	returnErr := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("wrong method"))
			return
		}
		if r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", key) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("wrong auth key"))
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if len(b) != len(receiveData) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fmt.Sprintf("wrong length received. %d != %d", len(b), len(receiveData))))
			return
		}
		for i, bt := range b {
			if bt != receiveData[i] {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(fmt.Sprintf("wrong byte at index %d", i)))
				return
			}
		}

		if !returnErr {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	endpoint, err := NewCollectorEndpoint(srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}

	err = endpoint.Submit(bytes.NewBuffer(receiveData))
	if err != nil {
		t.Fatal(err)
	}

	// Also test that errors bubble up
	returnErr = true
	err = endpoint.Submit(bytes.NewBuffer(receiveData))
	if err == nil {
		t.Fatal(err)
	}

	// ... and unplanned errors from the server (payload mismatch in this case)
	err = endpoint.Submit(bytes.NewBuffer(make([]byte, 4)))
	if err == nil {
		t.Fatal(err)
	}
}
