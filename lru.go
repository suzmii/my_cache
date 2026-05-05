package main

import (
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

// cachedNow 用原子变量缓存当前时间，后台 goroutine 每 100ms 刷新一次。
// 对缓存过期场景（秒级精度）完全够用，避免了 time.Now() 的系统调用开销。
var cachedNow atomic.Int64

func init() {
	cachedNow.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for t := range ticker.C {
			cachedNow.Store(t.UnixNano())
		}
	}()
}

func now() time.Time {
	return time.Unix(0, cachedNow.Load())
}

// refreshNow 强制刷新缓存时间，仅用于测试。
func refreshNow() {
	cachedNow.Store(time.Now().UnixNano())
}

type entry[K comparable, V any] struct {
	key      K
	value    V
	prev     *entry[K, V]
	next     *entry[K, V]
	deadline time.Time
	idx      int // 堆中的索引，-1 表示不在堆中
}

type expireHeap[K comparable, V any] []*entry[K, V]

func (e *expireHeap[K, V]) Pop() any {
	result := (*e)[len(*e)-1]
	result.idx = -1
	*e = (*e)[:len(*e)-1]
	return result
}

func (e *expireHeap[K, V]) Push(x any) {
	v := x.(*entry[K, V])
	v.idx = len(*e)
	*e = append(*e, v)
}

func (e expireHeap[K, V]) Len() int { return len(e) }

func (e expireHeap[K, V]) Less(i int, j int) bool { return e[i].deadline.Before(e[j].deadline) }

func (e expireHeap[K, V]) Swap(i int, j int) {
	e[i], e[j] = e[j], e[i]
	e[i].idx, e[j].idx = i, j
}

type LRU[K comparable, V any] struct {
	lock sync.Mutex
	size int
	m    map[K]*entry[K, V] // 哈希表
	h    expireHeap[K, V]
	head *entry[K, V]
	tail *entry[K, V]

	zero V
}

func NewLRU[K comparable, V any](size int) LRU[K, V] {
	head := &entry[K, V]{}
	tail := &entry[K, V]{}
	head.next = tail
	tail.prev = head

	if size < 1 {
		size = 1
	}

	return LRU[K, V]{
		size: size,
		m:    make(map[K]*entry[K, V]),
		h:    make(expireHeap[K, V], 0, size),
		head: head,
		tail: tail,
	}
}

func (l *LRU[K, V]) linkedListPushFront(e *entry[K, V]) {
	e.prev = l.head
	e.next = l.head.next
	l.head.next.prev = e
	l.head.next = e
}

func (l *LRU[K, V]) linkedListRemove(e *entry[K, V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
}

func (l *LRU[K, V]) linkedListMoveToFront(e *entry[K, V]) {
	l.linkedListRemove(e)
	l.linkedListPushFront(e)
}

// Set 写入无 TTL 的 key。大多数场景走这个路径，完全不触碰过期堆。
func (l *LRU[K, V]) Set(k K, v V) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if e, ok := l.m[k]; ok {
		e.value = v
		e.deadline = time.Time{}
		l.linkedListMoveToFront(e)
		if e.idx >= 0 {
			heap.Remove(&l.h, e.idx)
			e.idx = -1
		}
		return
	}

	var e *entry[K, V]
	if len(l.m) >= l.size {
		e = l.evictOne()
	} else {
		e = &entry[K, V]{}
	}

	e.key = k
	e.value = v
	e.deadline = time.Time{}
	e.idx = -1
	l.linkedListPushFront(e)
	l.m[k] = e
}

// SetWithTTL 写入带过期时间的 key。
func (l *LRU[K, V]) SetWithTTL(k K, v V, deadline time.Time) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if e, ok := l.m[k]; ok {
		e.value = v
		e.deadline = deadline
		l.linkedListMoveToFront(e)
		if e.idx >= 0 {
			heap.Fix(&l.h, e.idx)
		} else {
			heap.Push(&l.h, e)
		}
		return
	}

	var e *entry[K, V]
	if len(l.m) >= l.size {
		e = l.evictOne()
	} else {
		e = &entry[K, V]{}
	}

	e.key = k
	e.value = v
	e.deadline = deadline
	l.linkedListPushFront(e)
	heap.Push(&l.h, e)
	l.m[k] = e
}

func (l *LRU[K, V]) Get(k K) (V, bool) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if v, ok := l.m[k]; ok {
		if v.idx >= 0 && now().After(v.deadline) {
			l.removeEntry(v)
			return l.zero, false
		}
		l.linkedListMoveToFront(v)
		return v.value, ok
	}
	return l.zero, false
}

func (l *LRU[K, V]) removeEntry(e *entry[K, V]) {
	l.linkedListRemove(e)
	if e.idx >= 0 {
		heap.Remove(&l.h, e.idx)
	}
	delete(l.m, e.key)
}

// evictOne 驱逐一个元素：优先找已过期的，否则取 LRU 末尾。
func (l *LRU[K, V]) evictOne() *entry[K, V] {
	if len(l.h) > 0 && now().After(l.h[0].deadline) {
		e := l.h[0]
		l.removeEntry(e)
		return e
	}
	e := l.tail.prev
	l.removeEntry(e)
	return e
}
