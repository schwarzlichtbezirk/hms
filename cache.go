package hms

import (
	"sync"
)

type kvpair[K comparable, T any] struct {
	key K
	val T
}

type RWMap[K comparable, T any] struct {
	m   map[K]T
	mux sync.RWMutex
}

func (rwm *RWMap[K, T]) Init(c int) {
	if c < 8 {
		c = 8
	}
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	rwm.m = make(map[K]T, c)
}

func (rwm *RWMap[K, T]) Len() int {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	return len(rwm.m)
}

func (rwm *RWMap[K, T]) Has(key K) (ok bool) {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	_, ok = rwm.m[key]
	return
}

func (rwm *RWMap[K, T]) Get(key K) (ret T, ok bool) {
	rwm.mux.RLock()
	defer rwm.mux.RUnlock()
	ret, ok = rwm.m[key]
	return
}

func (rwm *RWMap[K, T]) Set(key K, val T) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	rwm.m[key] = val
}

func (rwm *RWMap[K, T]) Delete(key K) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	delete(rwm.m, key)
}

func (rwm *RWMap[K, T]) GetAndDelete(key K) (ret T, ok bool) {
	rwm.mux.Lock()
	defer rwm.mux.Unlock()
	if ret, ok = rwm.m[key]; ok {
		delete(rwm.m, key)
	}
	return
}

func (rwm *RWMap[K, T]) Range(f func(K, T) bool) {
	var buf []kvpair[K, T]
	func() {
		rwm.mux.RLock()
		defer rwm.mux.RUnlock()
		buf = make([]kvpair[K, T], len(rwm.m))
		var i int
		for k, v := range rwm.m {
			buf[i].key, buf[i].val = k, v
			i++
		}
	}()
	for _, pair := range buf {
		if !f(pair.key, pair.val) {
			return
		}
	}
}

type Cache[K comparable, T any] struct {
	seq []kvpair[K, T]
	idx map[K]int
	efn func(K, T)
	mux sync.Mutex
}

func NewCache[K comparable, T any]() *Cache[K, T] {
	return &Cache[K, T]{
		idx: map[K]int{},
	}
}

func (c *Cache[K, T]) OnRemove(efn func(K, T)) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.efn = efn
}

func (c *Cache[K, T]) Len() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return len(c.seq)
}

func (c *Cache[K, T]) Has(key K) (ok bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_, ok = c.idx[key]
	return
}

func (c *Cache[K, T]) Peek(key K) (ret T, ok bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	var n int
	if n, ok = c.idx[key]; ok {
		ret = c.seq[n].val
	}
	return
}

func (c *Cache[K, T]) Get(key K) (ret T, ok bool) {
	var n int

	c.mux.Lock()
	defer c.mux.Unlock()

	n, ok = c.idx[key]
	if ok {
		var pair = c.seq[n]
		ret = pair.val
		copy(c.seq[n:], c.seq[n+1:])
		c.seq[len(c.seq)-1] = pair
		for i := n; i < len(c.seq); i++ {
			c.idx[c.seq[i].key] = i
		}
	}
	return
}

func (c *Cache[K, T]) Poke(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.idx[key]
	if ok {
		c.seq[n].val = val
	} else {
		c.idx[key] = len(c.seq)
		c.seq = append(c.seq, kvpair[K, T]{
			key: key,
			val: val,
		})
	}
}

func (c *Cache[K, T]) Set(key K, val T) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n, ok = c.idx[key]
	if ok {
		var pair = c.seq[n]
		pair.val = val
		copy(c.seq[n:], c.seq[n+1:])
		c.seq[len(c.seq)-1] = pair
		for i := n; i < len(c.seq); i++ {
			c.idx[c.seq[i].key] = i
		}
	} else {
		c.idx[key] = len(c.seq)
		c.seq = append(c.seq, kvpair[K, T]{
			key: key,
			val: val,
		})
	}
}

func (c *Cache[K, T]) Remove(key K) (ok bool) {
	var n int

	c.mux.Lock()
	defer c.mux.Unlock()

	n, ok = c.idx[key]
	if ok {
		var pair = c.seq[n]
		if c.efn != nil {
			c.efn(pair.key, pair.val)
		}
		delete(c.idx, key)
		copy(c.seq[n:], c.seq[n+1:])
		c.seq = c.seq[:len(c.seq)-1]
		for i := n; i < len(c.seq); i++ {
			c.idx[c.seq[i].key] = i
		}
	}
	return
}

func (c *Cache[K, T]) Range(f func(K, T) bool) {
	c.mux.Lock()
	var s = append([]kvpair[K, T]{}, c.seq...) // make non-nil copy
	c.mux.Unlock()

	for _, pair := range s {
		if !f(pair.key, pair.val) {
			return
		}
	}
}

// Until removes first some entries from cache until given func returns true.
func (c *Cache[K, T]) Until(f func(K, T) bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var n = 0
	if c.efn != nil {
		for _, pair := range c.seq {
			if f(pair.key, pair.val) {
				c.efn(pair.key, pair.val)
				delete(c.idx, pair.key)
				n++
			} else {
				break
			}
		}
	} else {
		for _, pair := range c.seq {
			if f(pair.key, pair.val) {
				delete(c.idx, pair.key)
				n++
			} else {
				break
			}
		}
	}
	c.seq = c.seq[n:]
}

// Free removes n first entries from cache.
func (c *Cache[K, T]) Free(n int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if n <= 0 {
		return
	}
	if n >= len(c.seq) {
		if c.efn != nil {
			for _, pair := range c.seq {
				c.efn(pair.key, pair.val)
			}
		}
		c.idx = map[K]int{}
		c.seq = nil
		return
	}

	if c.efn != nil {
		for i := 0; i < n; i++ {
			c.efn(c.seq[i].key, c.seq[i].val)
			delete(c.idx, c.seq[i].key)
		}
	} else {
		for i := 0; i < n; i++ {
			delete(c.idx, c.seq[i].key)
		}
	}
	c.seq = c.seq[n:]
}

// ToLimit brings cache to limited count of entries.
func (c *Cache[K, T]) ToLimit(limit int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if limit >= len(c.seq) {
		return
	}
	if limit <= 0 {
		if c.efn != nil {
			for _, pair := range c.seq {
				c.efn(pair.key, pair.val)
			}
		}
		c.idx = map[K]int{}
		c.seq = nil
		return
	}

	var n = len(c.seq) - limit
	if c.efn != nil {
		for i := 0; i < n; i++ {
			c.efn(c.seq[i].key, c.seq[i].val)
			delete(c.idx, c.seq[i].key)
		}
	} else {
		for i := 0; i < n; i++ {
			delete(c.idx, c.seq[i].key)
		}
	}
	c.seq = c.seq[n:]
}

// Sizer is interface that determine structure size itself.
type Sizer interface {
	Size() int64
}

// CacheSize returns size of given cache.
func CacheSize[K comparable, T Sizer](cache *Cache[K, T]) (size int64) {
	cache.Range(func(key K, val T) bool {
		size += val.Size()
		return true
	})
	return
}

type Bimap[K comparable, T comparable] struct {
	dir map[K]T // direct order
	rev map[T]K // reverse order
	mux sync.RWMutex
}

func NewBimap[K comparable, T comparable]() *Bimap[K, T] {
	return &Bimap[K, T]{
		dir: map[K]T{},
		rev: map[T]K{},
	}
}

func (m *Bimap[K, T]) Len() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.dir)
}

func (m *Bimap[K, T]) GetDir(key K) (val T, ok bool) {
	m.mux.RLock()
	val, ok = m.dir[key]
	m.mux.RUnlock()
	return
}

func (m *Bimap[K, T]) GetRev(val T) (key K, ok bool) {
	m.mux.RLock()
	key, ok = m.rev[val]
	m.mux.RUnlock()
	return
}

func (m *Bimap[K, T]) Set(key K, val T) {
	m.mux.Lock()
	m.dir[key] = val
	m.rev[val] = key
	m.mux.Unlock()
}

func (m *Bimap[K, T]) DeleteDir(key K) (ok bool) {
	var val T
	m.mux.Lock()
	if val, ok = m.dir[key]; ok {
		delete(m.dir, key)
		delete(m.rev, val)
	}
	m.mux.Unlock()
	return
}

func (m *Bimap[K, T]) DeleteRev(val T) (ok bool) {
	var key K
	m.mux.Lock()
	if key, ok = m.rev[val]; ok {
		delete(m.dir, key)
		delete(m.rev, val)
	}
	m.mux.Unlock()
	return
}

// The End.
