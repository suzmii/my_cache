package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Body buffer pool — 复用 io.ReadAll 的缓冲区，减少分配
// ---------------------------------------------------------------------------

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// ---------------------------------------------------------------------------
// 预分配的错误响应体，避免每次 http.Error 做 string → []byte 转换
// ---------------------------------------------------------------------------

var (
	errBodyInvalidPath      = []byte("invalid path: expected /xxx/yyy/key\n")
	errBodyNotFound         = []byte("not found\n")
	errBodyReadBody         = []byte("read body error\n")
	errBodyMethodNotAllowed = []byte("method not allowed\n")
)

func writeHTTPError(w http.ResponseWriter, code int, body []byte) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write(body)
}

// ---------------------------------------------------------------------------
// 路由缓存 — sync.Map 读路径无锁
// ---------------------------------------------------------------------------

var stores sync.Map

func getOrCreateCache(path string) *Cache[string, []byte] {
	if v, ok := stores.Load(path); ok {
		return v.(*Cache[string, []byte])
	}

	c := NewCache[string, []byte](128, GetFunc[string, []byte](func(_ string) ([]byte, bool) {
		return nil, false
	}))

	actual, loaded := stores.LoadOrStore(path, &c)
	if loaded {
		return actual.(*Cache[string, []byte])
	}
	return &c
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

func handler(w http.ResponseWriter, r *http.Request) {
	idx := strings.LastIndex(r.URL.Path, "/")
	if idx < 0 || idx == len(r.URL.Path)-1 {
		writeHTTPError(w, http.StatusBadRequest, errBodyInvalidPath)
		return
	}

	path := r.URL.Path[:idx+1]
	key := r.URL.Path[idx+1:]

	c := getOrCreateCache(path)

	switch r.Method {
	case http.MethodGet:
		v, ok := c.Get(key)
		if !ok {
			writeHTTPError(w, http.StatusNotFound, errBodyNotFound)
			return
		}
		w.Write(v)

	case http.MethodPost, http.MethodPut:
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			bufPool.Put(buf)
			writeHTTPError(w, http.StatusInternalServerError, errBodyReadBody)
			return
		}
		// 拷贝一份存入缓存，buffer 归还池中
		body := make([]byte, buf.Len())
		copy(body, buf.Bytes())
		bufPool.Put(buf)

		c.Set(key, body)
		w.WriteHeader(http.StatusOK)

	default:
		writeHTTPError(w, http.StatusMethodNotAllowed, errBodyMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
