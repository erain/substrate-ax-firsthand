// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func request(t *testing.T, c *counter) map[string]int {
	t.Helper()
	response := httptest.NewRecorder()
	c.ServeHTTP(response, httptest.NewRequest("POST", "/", nil))
	if response.Code != 200 {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
	}
	var out map[string]int
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMemoryAndFileAreDifferentState(t *testing.T) {
	dir := t.TempDir()
	c, err := newCounter(dir)
	if err != nil {
		t.Fatal(err)
	}
	for count := 1; count <= 2; count++ {
		got := request(t, c)
		if got["memoryCount"] != count || got["fileCount"] != count {
			t.Fatal(got)
		}
	}
	fresh, err := newCounter(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := request(t, fresh)
	if got["memoryCount"] != 1 || got["fileCount"] != 3 {
		t.Fatal(got)
	}
}

func TestHTTPAndCorruptState(t *testing.T) {
	dir := t.TempDir()
	c, err := newCounter(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		code         int
	}{{"GET", "/readyz", 200}, {"GET", "/", 405}, {"POST", "/missing", 404}} {
		w := httptest.NewRecorder()
		c.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code {
			t.Fatal(w.Code)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "count.txt"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := newCounter(dir); err == nil {
		t.Fatal("accepted bad saved state")
	}
}

func TestConcurrentRequests(t *testing.T) {
	c, err := newCounter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			w := httptest.NewRecorder()
			c.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
			if w.Code != 200 {
				t.Errorf("HTTP %d", w.Code)
			}
		})
	}
	wg.Wait()
	got := request(t, c)
	if got["memoryCount"] != 11 || got["fileCount"] != 11 {
		t.Fatal(got)
	}
}
