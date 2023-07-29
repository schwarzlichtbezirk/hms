package hms

import (
	"sync"
)

type kvcell[K comparable, T any] struct {
	key K
	val T
}

type Cache[K comparable, T any] struct {
	s   []kvcell[K, T]
	m   map[K]int
	ef  func(K, T)
	mux sync.Mutex
}

func NewCache[K comparable, T any]() *Cache[K, T] {
	return &Cache[K, T]{
		m: map[K]int{},
	}
}

func (c *Cache[K, T]) OnRemove(ef func(K, T)) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.ef = ef
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
		ret = c.s[n].val
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
		ret = cell.val
		copy(c.s[n:], c.s[n+1:])
		c.s[len(c.s)-1] = cell
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	}
	return
}

func (c *Cache[K, T]) Poke(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.m[key]
	if ok {
		c.s[n].val = val
	} else {
		c.m[key] = len(c.s)
		c.s = append(c.s, kvcell[K, T]{
			key: key,
			val: val,
		})
	}
}

func (c *Cache[K, T]) Set(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.m[key]
	if ok {
		var cell = c.s[n]
		cell.val = val
		copy(c.s[n:], c.s[n+1:])
		c.s[len(c.s)-1] = cell
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	} else {
		c.m[key] = len(c.s)
		c.s = append(c.s, kvcell[K, T]{
			key: key,
			val: val,
		})
	}
}

func (c *Cache[K, T]) Remove(key K) (ok bool) {
	var n int

	c.mux.Lock()
	defer c.mux.Unlock()

	n, ok = c.m[key]
	if ok {
		var cell = c.s[n]
		if c.ef != nil {
			c.ef(cell.key, cell.val)
		}
		delete(c.m, key)
		copy(c.s[n:], c.s[n+1:])
		c.s = c.s[:len(c.s)-1]
		for i := n; i < len(c.s); i++ {
			c.m[c.s[i].key] = i
		}
	}
	return
}

func (c *Cache[K, T]) Range(f func(K, T) bool) {
	c.mux.Lock()
	var s = append([]kvcell[K, T]{}, c.s...) // make non-nil copy
	c.mux.Unlock()

	for _, cell := range s {
		if !f(cell.key, cell.val) {
			return
		}
	}
}

// Until removes first some entries from cache until given func returns true.
func (c *Cache[K, T]) Until(f func(K, T) bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n = 0
	if c.ef != nil {
		for _, cell := range c.s {
			if f(cell.key, cell.val) {
				c.ef(cell.key, cell.val)
				delete(c.m, cell.key)
				n++
			} else {
				break
			}
		}
	} else {
		for _, cell := range c.s {
			if f(cell.key, cell.val) {
				delete(c.m, cell.key)
				n++
			} else {
				break
			}
		}
	}
	c.s = c.s[n:]
}

// Free removes n first entries from cache.
func (c *Cache[K, T]) Free(n int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if n <= 0 {
		return
	}
	if n >= len(c.s) {
		if c.ef != nil {
			for _, cell := range c.s {
				c.ef(cell.key, cell.val)
			}
		}
		c.m = map[K]int{}
		c.s = nil
		return
	}

	if c.ef != nil {
		for i := 0; i < n; i++ {
			c.ef(c.s[i].key, c.s[i].val)
			delete(c.m, c.s[i].key)
		}
	} else {
		for i := 0; i < n; i++ {
			delete(c.m, c.s[i].key)
		}
	}
	c.s = c.s[n:]
}

// ToLimit brings cache to limited count of entries.
func (c *Cache[K, T]) ToLimit(limit int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if limit >= len(c.s) {
		return
	}
	if limit <= 0 {
		if c.ef != nil {
			for _, cell := range c.s {
				c.ef(cell.key, cell.val)
			}
		}
		c.m = map[K]int{}
		c.s = nil
		return
	}

	var n = len(c.s) - limit
	if c.ef != nil {
		for i := 0; i < n; i++ {
			c.ef(c.s[i].key, c.s[i].val)
			delete(c.m, c.s[i].key)
		}
	} else {
		for i := 0; i < n; i++ {
			delete(c.m, c.s[i].key)
		}
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
