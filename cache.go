package hms

import (
	"sync"
)

type kvcell[K comparable, T any] struct {
	key   K
	value T
}

type Cache[K comparable, T any] struct {
	s   []kvcell[K, T]
	m   map[K]int
	mux sync.Mutex
}

func NewCache[K comparable, T any]() *Cache[K, T] {
	return &Cache[K, T]{
		m: map[K]int{},
	}
}

func (c *Cache[K, T]) Len() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return len(c.s)
}

func (c *Cache[K, T]) Has(key K) (ok bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, ok = c.m[key]
	return
}

func (c *Cache[K, T]) Peek(key K) (ret T, ok bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	var n int
	if n, ok = c.m[key]; ok {
		ret = c.s[n].value
	}
	return
}

func (c *Cache[K, T]) Get(key K) (ret T, ok bool) {
	var n int

	c.mux.Lock()
	defer c.mux.Unlock()

	n, ok = c.m[key]
	if ok {
		var cell = c.s[n]
		ret = cell.value
		copy(c.s[n:], c.s[n+1:])
		c.s[len(c.s)-1] = cell
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	}
	return
}

func (c *Cache[K, T]) Push(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.m[key]
	if ok {
		c.s[n].value = val
	} else {
		c.m[key] = len(c.s)
		c.s = append(c.s, kvcell[K, T]{
			key:   key,
			value: val,
		})
	}
}

func (c *Cache[K, T]) Set(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.m[key]
	if ok {
		var cell = c.s[n]
		cell.value = val
		copy(c.s[n:], c.s[n+1:])
		c.s[len(c.s)-1] = cell
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	} else {
		c.m[key] = len(c.s)
		c.s = append(c.s, kvcell[K, T]{
			key:   key,
			value: val,
		})
	}
}

func (c *Cache[K, T]) Remove(key K) (ok bool) {
	var n int

	c.mux.Lock()
	defer c.mux.Unlock()

	n, ok = c.m[key]
	if ok {
		delete(c.m, key)
		copy(c.s[n:], c.s[n+1:])
		c.s = c.s[:len(c.s)-1]
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	}
	return
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

	if n <= 0 {
		return
	}
	if n >= len(c.s) {
		c.s = nil
		c.m = map[K]int{}
		return
	}

	for i := 0; i < n; i++ {
		delete(c.m, c.s[i].key)
	}
	c.s = c.s[n:]
}

func (c *Cache[K, T]) ToLimit(limit int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if limit >= len(c.s) {
		return
	}
	if limit <= 0 {
		c.s = nil
		c.m = map[K]int{}
		return
	}

	var n = len(c.s) - limit
	for i := 0; i < n; i++ {
		delete(c.m, c.s[i].key)
	}
	c.s = c.s[n:]
}

type Bimap[K comparable, T comparable] struct {
	direct  map[K]T
	reverse map[T]K
	mux     sync.RWMutex
}

func NewBimap[K comparable, T comparable]() *Bimap[K, T] {
	return &Bimap[K, T]{
		direct:  map[K]T{},
		reverse: map[T]K{},
	}
}

func (m *Bimap[K, T]) Len() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.direct)
}

func (m *Bimap[K, T]) GetDir(key K) (val T, ok bool) {
	m.mux.RLock()
	val, ok = m.direct[key]
	m.mux.RUnlock()
	return
}

func (m *Bimap[K, T]) GetRev(val T) (key K, ok bool) {
	m.mux.RLock()
	key, ok = m.reverse[val]
	m.mux.RUnlock()
	return
}

func (m *Bimap[K, T]) Set(key K, val T) {
	m.mux.Lock()
	m.direct[key] = val
	m.reverse[val] = key
	m.mux.Unlock()
}

func (m *Bimap[K, T]) DeleteDir(key K) (ok bool) {
	var val T
	m.mux.Lock()
	if val, ok = m.direct[key]; ok {
		delete(m.direct, key)
		delete(m.reverse, val)
	}
	m.mux.Unlock()
	return
}

func (m *Bimap[K, T]) DeleteRev(val T) (ok bool) {
	var key K
	m.mux.Lock()
	if key, ok = m.reverse[val]; ok {
		delete(m.direct, key)
		delete(m.reverse, val)
	}
	m.mux.Unlock()
	return
}

// The End.
