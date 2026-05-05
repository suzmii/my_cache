package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

var (
	mu     sync.RWMutex
	stores = make(map[string]*Cache[string, []byte])
)

func getOrCreateCache(path string) *Cache[string, []byte] {
	// Fast path: read lock to check if cache already exists
	mu.RLock()
	if c, ok := stores[path]; ok {
		mu.RUnlock()
		return c
	}
	mu.RUnlock()

	// Slow path: write lock to create a new cache
	mu.Lock()
	// Double-check: another goroutine may have created it between the unlock and lock
	if c, ok := stores[path]; ok {
		mu.Unlock()
		return c
	}

	c := NewCache(128, GetFunc[string, []byte](func(_ string) ([]byte, bool) {
		return nil, false
	}))
	stores[path] = &c
	mu.Unlock()
	return &c
}

func handler(w http.ResponseWriter, r *http.Request) {
	idx := strings.LastIndex(r.URL.Path, "/")
	if idx < 0 || idx == len(r.URL.Path)-1 {
		http.Error(w, "invalid path: expected /xxx/yyy/key", http.StatusBadRequest)
		return
	}

	path := r.URL.Path[:idx+1]
	key := r.URL.Path[idx+1:]

	c := getOrCreateCache(path)

	switch r.Method {
	case http.MethodGet:
		v, ok := c.Get(key)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(v)

	case http.MethodPost, http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body error", http.StatusInternalServerError)
			return
		}
		c.Set(key, body)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
