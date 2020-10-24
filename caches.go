package hms

import (
	"crypto/rand"
	"encoding/base32"
	"os"
	"sync"

	"github.com/bluele/gcache"
)

// gcaches
var (
	// Files properties cache.
	// Key - system path, value - file property struct.
	propcache gcache.Cache

	// Thumbnails cache.
	// Key - path unique ID, value - thumbnail image.
	thumbcache gcache.Cache
)

// Unlimited cache with puid/syspath and syspath/puid values.
type KeyThumbCache struct {
	keypath map[string]string
	pathkey map[string]string
	mux     sync.RWMutex
}

// Returns cached PUID for specified system path.
func (c *KeyThumbCache) PUID(syspath string) (puid string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	puid, ok = c.pathkey[syspath]
	return
}

// Returns cached system path of specified PUID (path unique identifier).
func (c *KeyThumbCache) Path(puid string) (syspath string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	syspath, ok = c.keypath[puid]
	return
}

// Returns cached PUID for specified system path, or make it and put into cache.
func (c *KeyThumbCache) Cache(syspath string) string {
	var puid string
	var ok bool

	c.mux.Lock()
	defer c.mux.Unlock()

	if puid, ok = c.pathkey[syspath]; ok {
		return puid
	}

	// generate path unique ID
	for ok = true; ok; _, ok = c.keypath[puid] {
		var buf [10]byte
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}
		puid = idenc.EncodeToString(buf[:])
	}

	c.pathkey[syspath] = puid
	c.keypath[puid] = syspath
	return puid
}

// Instance of unlimited cache with PUID<=>syspath pairs.
var pathcache = KeyThumbCache{
	keypath: map[string]string{},
	pathkey: map[string]string{},
}

// Produce base32 string representation of given random bytes slice.
var idenc = base32.HexEncoding.WithPadding(base32.NoPadding)

// Prepares caches depends of previously loaded configuration.
func initcaches() {
	// init properties cache
	propcache = gcache.New(cfg.PropCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath = key.(string)
			var fi os.FileInfo
			if fi, err = os.Stat(syspath); err != nil {
				return
			}
			ret = MakeProp(syspath, fi)
			return
		}).
		Build()

	// init thumbnails cache
	thumbcache = gcache.New(cfg.ThumbCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = pathcache.Path(key.(string))
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var cp interface{}
			if cp, err = propcache.Get(syspath); err != nil {
				return // can not get properties
			}
			var prop = cp.(Proper)
			if prop.NTmb() == TMB_reject {
				err = ErrNotThumb
				return // thumbnail rejected
			}

			var tmb *ThumbElem
			if tmb, err = FindTmb(prop, syspath); tmb != nil {
				prop.SetNTmb(TMB_cached)
				ret = tmb
			} else {
				prop.SetNTmb(TMB_reject)
			}
			return // ok
		}).
		Build()
}

// The End.
