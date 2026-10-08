// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type counter struct {
	mu     sync.Mutex
	memory int
	path   string
}

func newCounter(dir string) (*counter, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	c := &counter{path: filepath.Join(dir, "count.txt")}
	if _, err := c.readFile(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *counter) readFile() (int, error) {
	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid saved counter")
	}
	return n, nil
}

func (c *counter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/readyz" && r.Method == http.MethodGet {
		w.WriteHeader(200)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	fileCount, err := c.readFile()
	if err != nil {
		http.Error(w, "cannot read counter", 500)
		return
	}
	file, err := os.CreateTemp(filepath.Dir(c.path), ".count-*")
	if err != nil {
		http.Error(w, "cannot save counter", 500)
		return
	}
	defer os.Remove(file.Name())
	if _, err = fmt.Fprintln(file, fileCount+1); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(file.Name(), c.path)
	}
	if err != nil {
		http.Error(w, "cannot save counter", 500)
		return
	}
	c.memory++
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"memoryCount": c.memory, "fileCount": fileCount + 1})
}

func main() {
	c, err := newCounter("/data")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: ":80", Handler: c, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
