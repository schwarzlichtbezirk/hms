package hms

import (
	"sync"
)

type kvcell[K comparable, T any] struct {
	key   K
	value T
}

type Cache[K comparable, T any] struct {
	s     []kvcell[K, T]
	m     map[K]int
	mux   sync.Mutex
	limit int
}

func NewCache[K comparable, T any](limit int) *Cache[K, T] {
	return &Cache[K, T]{
		m:     map[K]int{},
		limit: limit,
	}
}

func (c *Cache[K, T]) Get(key K) (ret T, ok bool) {
	var i int

	c.mux.Lock()
	defer c.mux.Unlock()

	i, ok = c.m[key]
	if ok {
		var cell = c.s[i]
		ret = cell.value
		copy(c.s[i:], c.s[i+1:])
		c.s[len(c.s)-1] = cell
	}
	return
}

func (c *Cache[K, T]) Set(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var i, ok = c.m[key]
	if ok {
		var cell = c.s[i]
		cell.value = val
		copy(c.s[i:], c.s[i+1:])
		c.s[len(c.s)-1] = cell
	} else {
		c.s = append(c.s, kvcell[K, T]{
			key:   key,
			value: val,
		})
		c.tolimit()
	}
}

func (c *Cache[K, T]) Enum(f func(K, T) bool) {
	var s = make([]kvcell[K, T], len(c.s))
	c.mux.Lock()
	copy(s, c.s)
	c.mux.Unlock()

	for _, cell := range s {
		if !f(cell.key, cell.value) {
			return
		}
	}
}

func (c *Cache[K, T]) Free(n int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if n < 0 || n >= len(c.s) {
		c.s = nil
		c.m = map[K]int{}
		return
	}

	for i := 0; i < n; i++ {
		delete(c.m, c.s[i].key)
	}
	c.s = c.s[n:]
}

func (c *Cache[K, T]) Len() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return len(c.s)
}

func (c *Cache[K, T]) Limit() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.limit
}

func (c *Cache[K, T]) SetLimit(limit int) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.limit = limit
	c.tolimit()
}

func (c *Cache[K, T]) tolimit() {
	if c.limit > 0 {
		var n = len(c.s) - c.limit
		if n > 0 {
			for i := 0; i < n; i++ {
				delete(c.m, c.s[i].key)
			}
			c.s = c.s[n:]
		}
	}
}

// The End.
