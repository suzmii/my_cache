package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerGetNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test/key1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestHandlerPostAndGet(t *testing.T) {
	// First, POST a value
	body := "hello world"
	req := httptest.NewRequest(http.MethodPost, "/test/key1", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d on POST, got %d", http.StatusOK, resp.StatusCode)
	}

	// Then, GET the value back
	req = httptest.NewRequest(http.MethodGet, "/test/key1", nil)
	w = httptest.NewRecorder()

	handler(w, req)

	resp = w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d on GET, got %d", http.StatusOK, resp.StatusCode)
	}

	got := w.Body.String()
	if got != body {
		t.Fatalf("expected body %q, got %q", body, got)
	}
}

func TestHandlerPutAndGet(t *testing.T) {
	// PUT a value
	body := "put value"
	req := httptest.NewRequest(http.MethodPut, "/test/key2", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d on PUT, got %d", http.StatusOK, resp.StatusCode)
	}

	// GET the value
	req = httptest.NewRequest(http.MethodGet, "/test/key2", nil)
	w = httptest.NewRecorder()

	handler(w, req)

	resp = w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d on GET, got %d", http.StatusOK, resp.StatusCode)
	}

	got := w.Body.String()
	if got != body {
		t.Fatalf("expected body %q, got %q", body, got)
	}
}

func TestHandlerOverwrite(t *testing.T) {
	// POST initial value
	req := httptest.NewRequest(http.MethodPost, "/test/key3", strings.NewReader("first"))
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	resp.Body.Close()

	// POST again to overwrite
	req = httptest.NewRequest(http.MethodPost, "/test/key3", strings.NewReader("second"))
	w = httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	resp.Body.Close()

	// GET should return the overwritten value
	req = httptest.NewRequest(http.MethodGet, "/test/key3", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	defer resp.Body.Close()

	got := w.Body.String()
	if got != "second" {
		t.Fatalf("expected body %q after overwrite, got %q", "second", got)
	}
}

func TestHandlerInvalidPathNoKey(t *testing.T) {
	// Path with no key (e.g., "/test/")
	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d for path with no key, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestHandlerPathSingleSegment(t *testing.T) {
	// Path with a single segment like "/test" is treated as cache path "/" with key "test"
	// First POST to store a value
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("single-segment"))
	w := httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	// Then GET to retrieve it
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if w.Body.String() != "single-segment" {
		t.Fatalf("expected 'single-segment', got %q", w.Body.String())
	}

	// A different key under the same root cache path "/" should also work
	req = httptest.NewRequest(http.MethodPost, "/another", strings.NewReader("another-value"))
	w = httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/another", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	defer resp.Body.Close()

	if w.Body.String() != "another-value" {
		t.Fatalf("expected 'another-value', got %q", w.Body.String())
	}
}

func TestHandlerInvalidPathRoot(t *testing.T) {
	// Root path "/"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d for root path, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/test/key1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d for DELETE, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestHandlerMethodHead(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "/test/key1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d for HEAD, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestHandlerMultiplePaths(t *testing.T) {
	// POST to different paths
	paths := []struct {
		path string
		body string
	}{
		{"/users/alice", "Alice data"},
		{"/users/bob", "Bob data"},
		{"/posts/first", "First post"},
	}

	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p.path, strings.NewReader(p.body))
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d for POST %s, got %d", http.StatusOK, p.path, resp.StatusCode)
		}
	}

	// GET each one back
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p.path, nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d for GET %s, got %d", http.StatusOK, p.path, resp.StatusCode)
		}

		got := w.Body.String()
		if got != p.body {
			t.Fatalf("for path %s: expected body %q, got %q", p.path, p.body, got)
		}
	}
}

func TestHandlerSameKeyDifferentPaths(t *testing.T) {
	// Same key name but under different paths should be independent caches
	req := httptest.NewRequest(http.MethodPost, "/app1/key", strings.NewReader("app1-value"))
	w := httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	req = httptest.NewRequest(http.MethodPost, "/app2/key", strings.NewReader("app2-value"))
	w = httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	// GET from app1
	req = httptest.NewRequest(http.MethodGet, "/app1/key", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if w.Body.String() != "app1-value" {
		t.Fatalf("expected 'app1-value' for /app1/key, got %q", w.Body.String())
	}

	// GET from app2
	req = httptest.NewRequest(http.MethodGet, "/app2/key", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	resp.Body.Close()

	if w.Body.String() != "app2-value" {
		t.Fatalf("expected 'app2-value' for /app2/key, got %q", w.Body.String())
	}
}

func TestHandlerPostEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test/empty", strings.NewReader(""))
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d for POST with empty body, got %d", http.StatusOK, resp.StatusCode)
	}

	// GET should return empty
	req = httptest.NewRequest(http.MethodGet, "/test/empty", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	defer resp.Body.Close()

	got := w.Body.String()
	if got != "" {
		t.Fatalf("expected empty body, got %q", got)
	}
}

func TestHandlerPostBinaryBody(t *testing.T) {
	body := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	req := httptest.NewRequest(http.MethodPost, "/test/binary", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler(w, req)
	w.Result().Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/test/binary", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	got := w.Body.Bytes()
	if string(got) != string(body) {
		t.Fatalf("expected binary body %v, got %v", body, got)
	}
}

func TestHandlerNestedPath(t *testing.T) {
	// Deeply nested path like /a/b/c/d/key
	req := httptest.NewRequest(http.MethodPost, "/a/b/c/d/key", strings.NewReader("nested"))
	w := httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/a/b/c/d/key", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	if w.Body.String() != "nested" {
		t.Fatalf("expected 'nested', got %q", w.Body.String())
	}
}

func TestHandlerLargeBody(t *testing.T) {
	// Test with a larger body (1MB)
	body := strings.Repeat("x", 1024*1024)

	req := httptest.NewRequest(http.MethodPost, "/test/large", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler(w, req)
	w.Result().Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/test/large", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	got := w.Body.String()
	if got != body {
		t.Fatalf("expected large body of length %d, got length %d", len(body), len(got))
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkHandlerGet(b *testing.B) {
	// Seed a value first
	req := httptest.NewRequest(http.MethodPost, "/bench/get-key", strings.NewReader("bench-value"))
	w := httptest.NewRecorder()
	handler(w, req)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/bench/get-key", nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkHandlerGetNotFound(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/bench/missing", nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkHandlerPost(b *testing.B) {
	body := strings.Repeat("a", 1024) // 1KB payload

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/bench/post-key", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkHandlerPostSmallBody(b *testing.B) {
	body := "hello"

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/bench/small", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkHandlerGetParallel(b *testing.B) {
	// Seed a value
	req := httptest.NewRequest(http.MethodPost, "/bench/parallel-key", strings.NewReader("parallel-value"))
	w := httptest.NewRecorder()
	handler(w, req)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/bench/parallel-key", nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}
	})
}

func BenchmarkHandlerPostParallel(b *testing.B) {
	body := strings.Repeat("a", 1024)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Each goroutine uses a different key to avoid contention
			key := fmt.Sprintf("/bench/parallel-%d-%d", i, i)
			req := httptest.NewRequest(http.MethodPost, key, strings.NewReader(body))
			w := httptest.NewRecorder()
			handler(w, req)
			i++
		}
	})
}

func BenchmarkHandlerGetDifferentPaths(b *testing.B) {
	// Seed values under different paths
	paths := []string{"/users/1", "/posts/2", "/comments/3", "/tags/4"}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p, strings.NewReader("data"))
		w := httptest.NewRecorder()
		handler(w, req)
	}

	b.ResetTimer()
	for b.Loop() {
		for _, p := range paths {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}
	}
}

// ---------------------------------------------------------------------------
// Baseline benchmarks — 测量 httptest 本身的开销
// ---------------------------------------------------------------------------

func BenchmarkHttptestOverheadGet(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = httptest.NewRequest(http.MethodGet, "/bench/key", nil)
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
	}
}

func BenchmarkHttptestOverheadPost(b *testing.B) {
	body := "hello"
	b.ResetTimer()
	for b.Loop() {
		_ = httptest.NewRequest(http.MethodPost, "/bench/key", strings.NewReader(body))
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusOK)
	}
}
