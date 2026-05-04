package main

type Getter[K comparable, V any] interface {
	Get(k K) (V, bool)
}

var _ Getter[int, any] = GetFunc[int, any](func(_ int) (any, bool) { return nil, true })

type GetFunc[K comparable, V any] func(k K) (V, bool)

func (f GetFunc[K, V]) Get(k K) (V, bool) {
	return f(k)
}
