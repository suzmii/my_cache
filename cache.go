package main

type Cache[K comparable, V any] struct {
	size   int
	lru    LRU[K, V]
	getter Getter[K, V]
}

// TODO: bloom filter

func NewCache[K comparable, V any](size int, getter Getter[K, V]) Cache[K, V] {
	return Cache[K, V]{
		size:   size,
		lru:    NewLRU[K, V](size),
		getter: getter,
	}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	if v, ok := c.lru.Get(k); ok {
		return v, ok
	}

	if v, ok := c.getter.Get(k); ok {
		c.lru.Set(k, v)
		return v, ok
	}

	return c.lru.zero, false
}

func (c *Cache[K, V]) Set(k K, v V) {
	c.lru.Set(k, v)
}
