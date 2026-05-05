package main

import (
	"testing"
	"time"
)

func TestNewLRU(t *testing.T) {
	lru := NewLRU[int, string](3)
	if lru.size != 3 {
		t.Fatalf("expected size 3, got %d", lru.size)
	}
	if len(lru.m) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(lru.m))
	}
}

func TestSetAndGet(t *testing.T) {
	lru := NewLRU[string, int](10)

	lru.Set("a", 1)
	lru.Set("b", 2)
	lru.Set("c", 3)

	val, ok := lru.Get("a")
	if !ok {
		t.Fatal("expected key 'a' to exist")
	}
	if val != 1 {
		t.Fatalf("expected value 1, got %d", val)
	}

	val, ok = lru.Get("b")
	if !ok {
		t.Fatal("expected key 'b' to exist")
	}
	if val != 2 {
		t.Fatalf("expected value 2, got %d", val)
	}

	val, ok = lru.Get("c")
	if !ok {
		t.Fatal("expected key 'c' to exist")
	}
	if val != 3 {
		t.Fatalf("expected value 3, got %d", val)
	}
}

func TestGetNonExistentKey(t *testing.T) {
	lru := NewLRU[string, int](10)

	_, ok := lru.Get("nonexistent")
	if ok {
		t.Fatal("expected key 'nonexistent' to not exist")
	}
}

func TestGetZeroValue(t *testing.T) {
	lru := NewLRU[string, int](10)

	val, _ := lru.Get("nonexistent")
	if val != 0 {
		t.Fatalf("expected zero value 0, got %d", val)
	}
}

func TestCapacityEviction(t *testing.T) {
	lru := NewLRU[int, string](3)

	lru.Set(1, "one")
	lru.Set(2, "two")
	lru.Set(3, "three")

	// Cache is full; inserting a new element should evict the least recently used (1)
	lru.Set(4, "four")

	// Key 1 should have been evicted
	_, ok := lru.Get(1)
	if ok {
		t.Fatal("expected key 1 to be evicted")
	}

	// Keys 2, 3, 4 should still exist
	for _, k := range []int{2, 3, 4} {
		_, ok := lru.Get(k)
		if !ok {
			t.Fatalf("expected key %d to exist", k)
		}
	}
}

func TestLRUOrderEviction(t *testing.T) {
	lru := NewLRU[int, string](3)

	lru.Set(1, "one")
	lru.Set(2, "two")
	lru.Set(3, "three")

	// Access key 1 to make it most recently used
	lru.Get(1)

	// Now key 2 is the least recently used
	lru.Set(4, "four")

	// Key 2 should be evicted
	_, ok := lru.Get(2)
	if ok {
		t.Fatal("expected key 2 to be evicted")
	}

	// Keys 1, 3, 4 should remain
	for _, k := range []int{1, 3, 4} {
		_, ok := lru.Get(k)
		if !ok {
			t.Fatalf("expected key %d to exist", k)
		}
	}
}

func TestSetWithTTL_Expired(t *testing.T) {
	lru := NewLRU[string, int](10)

	// Set with a TTL that expires in 10 milliseconds
	lru.SetWithTTL("a", 1, time.Now().Add(10*time.Millisecond))

	// Should be available immediately
	val, ok := lru.Get("a")
	if !ok {
		t.Fatal("expected key 'a' to exist before expiration")
	}
	if val != 1 {
		t.Fatalf("expected value 1, got %d", val)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	refreshNow()

	// Should be expired now
	_, ok = lru.Get("a")
	if ok {
		t.Fatal("expected key 'a' to be expired")
	}
}

func TestSetWithTTL_FarFuture(t *testing.T) {
	lru := NewLRU[string, int](10)

	farFuture := time.Now().Add(24 * time.Hour)
	lru.SetWithTTL("a", 42, farFuture)

	val, ok := lru.Get("a")
	if !ok {
		t.Fatal("expected key 'a' to exist")
	}
	if val != 42 {
		t.Fatalf("expected value 42, got %d", val)
	}
}

func TestSetDefaultTTL(t *testing.T) {
	lru := NewLRU[string, int](10)

	lru.Set("a", 100)

	val, ok := lru.Get("a")
	if !ok {
		t.Fatal("expected key 'a' to exist")
	}
	if val != 100 {
		t.Fatalf("expected value 100, got %d", val)
	}
}

func TestExpiredEntryRemoved(t *testing.T) {
	lru := NewLRU[string, int](10)

	lru.SetWithTTL("a", 1, time.Now().Add(10*time.Millisecond))
	lru.Set("b", 2)

	time.Sleep(20 * time.Millisecond)
	refreshNow()

	// Accessing 'a' should detect expiration and remove it
	_, ok := lru.Get("a")
	if ok {
		t.Fatal("expected key 'a' to be expired")
	}

	// 'b' should still exist
	_, ok = lru.Get("b")
	if !ok {
		t.Fatal("expected key 'b' to exist")
	}
}

func TestEvictExpiredBeforeLRU(t *testing.T) {
	lru := NewLRU[int, string](2)

	// Insert two entries with short TTL
	lru.SetWithTTL(1, "one", time.Now().Add(10*time.Millisecond))
	lru.SetWithTTL(2, "two", time.Now().Add(10*time.Millisecond))

	time.Sleep(20 * time.Millisecond)
	refreshNow()

	// Insert a new entry, eviction should first remove expired entries
	lru.Set(3, "three")

	// Both 1 and 2 should be gone (expired), 3 should be in
	_, ok := lru.Get(1)
	if ok {
		t.Fatal("expected key 1 to be expired and removed")
	}
	_, ok = lru.Get(2)
	if ok {
		t.Fatal("expected key 2 to be expired and removed")
	}
	_, ok = lru.Get(3)
	if !ok {
		t.Fatal("expected key 3 to exist")
	}
}

func TestEmptyCache(t *testing.T) {
	lru := NewLRU[int, string](0)

	lru.Set(1, "one")
	_, ok := lru.Get(1)
	// With size 0, behavior depends on implementation; at minimum it should not panic
	_ = ok
}

func TestMultipleTypes(t *testing.T) {
	// Test with struct keys
	type Key struct {
		ID   int
		Name string
	}

	lru := NewLRU[Key, []byte](5)

	k1 := Key{ID: 1, Name: "foo"}
	k2 := Key{ID: 2, Name: "bar"}

	lru.Set(k1, []byte("hello"))
	lru.Set(k2, []byte("world"))

	val, ok := lru.Get(k1)
	if !ok {
		t.Fatal("expected key k1 to exist")
	}
	if string(val) != "hello" {
		t.Fatalf("expected 'hello', got '%s'", string(val))
	}

	val, ok = lru.Get(k2)
	if !ok {
		t.Fatal("expected key k2 to exist")
	}
	if string(val) != "world" {
		t.Fatalf("expected 'world', got '%s'", string(val))
	}
}

func TestSetOverwrite(t *testing.T) {
	lru := NewLRU[string, int](10)

	lru.Set("a", 1)
	lru.Set("a", 2)

	val, ok := lru.Get("a")
	if !ok {
		t.Fatal("expected key 'a' to exist")
	}
	if val != 2 {
		t.Fatalf("expected value 2 after overwrite, got %d", val)
	}
}

func TestConcurrentAccess(t *testing.T) {
	lru := NewLRU[int, int](100)

	done := make(chan bool)
	for i := range 100 {
		go func(id int) {
			for j := range 100 {
				lru.Set(id*100+j, id*100+j)
				lru.Get(id * 100)
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}
	// If we reach here without deadlock, concurrent access is safe
}

// ---------------------------------------------------------------------------
// LRU Benchmarks — focus on heap overhead at scale
// ---------------------------------------------------------------------------

func benchmarkLRUSet(size int, b *testing.B) {
	lru := NewLRU[int, int](size)
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.Set(i%size, i)
		i++
	}
}

func benchmarkLRUSetWithTTL(size int, b *testing.B) {
	lru := NewLRU[int, int](size)
	deadline := time.Now().Add(time.Hour)
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.SetWithTTL(i%size, i, deadline)
		i++
	}
}

func BenchmarkLRUSet_128(b *testing.B)  { benchmarkLRUSet(128, b) }
func BenchmarkLRUSet_1K(b *testing.B)   { benchmarkLRUSet(1000, b) }
func BenchmarkLRUSet_10K(b *testing.B)  { benchmarkLRUSet(10000, b) }
func BenchmarkLRUSet_100K(b *testing.B) { benchmarkLRUSet(100000, b) }

func BenchmarkLRUSetWithTTL_128(b *testing.B)  { benchmarkLRUSetWithTTL(128, b) }
func BenchmarkLRUSetWithTTL_1K(b *testing.B)   { benchmarkLRUSetWithTTL(1000, b) }
func BenchmarkLRUSetWithTTL_10K(b *testing.B)  { benchmarkLRUSetWithTTL(10000, b) }
func BenchmarkLRUSetWithTTL_100K(b *testing.B) { benchmarkLRUSetWithTTL(100000, b) }

// BenchmarkLRUSet_Evict 使用永不重复的 key，持续触发 eviction，测满缓存插入路径。
func BenchmarkLRUSet_Evict(b *testing.B) {
	lru := NewLRU[int, int](128)
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.Set(i, i)
		i++
	}
}

// BenchmarkLRUSetWithTTL_Evict 同上，带 TTL。
func BenchmarkLRUSetWithTTL_Evict(b *testing.B) {
	lru := NewLRU[int, int](128)
	deadline := time.Now().Add(time.Hour)
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.SetWithTTL(i, i, deadline)
		i++
	}
}

func BenchmarkLRUGet_Hit(b *testing.B) {
	lru := NewLRU[int, int](10000)
	for i := range 5000 {
		lru.Set(i, i)
	}
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.Get(i % 5000)
		i++
	}
}

func BenchmarkLRUGet_Miss(b *testing.B) {
	lru := NewLRU[int, int](10000)
	for i := range 5000 {
		lru.Set(i, i)
	}
	b.ResetTimer()
	i := 0
	for b.Loop() {
		lru.Get(i + 10000)
		i++
	}
}
