package hms

import (
	"crypto/hmac"
	"crypto/md5"
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
	// Key - thumbnail hash-key (syspath MD5-hash), value - thumbnail image.
	thumbcache gcache.Cache
)

// Unlimited cache with hash-key/syspath and syspath/hash-key values.
type KeyThumbCache struct {
	keypath map[string]string
	pathkey map[string]string
	mux     sync.RWMutex
}

// Returns cached hash-key for specified system path.
func (c *KeyThumbCache) Key(syspath string) (hash string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	hash, ok = c.pathkey[syspath]
	return
}

// Returns cached system path of specified hash-key.
func (c *KeyThumbCache) Path(hash string) (syspath string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	syspath, ok = c.keypath[hash]
	return
}

// Returns cached hash-key for specified system path, or make it and put into cache.
func (c *KeyThumbCache) Cache(syspath string) string {
	if hash, ok := c.Key(syspath); ok {
		return hash
	}

	var mac = hmac.New(md5.New, []byte(cfg.PathHashSalt))
	mac.Write([]byte(syspath))
	var h = mac.Sum(nil)
	var hash = keygen.EncodeToString(h[:])

	c.mux.Lock()
	defer c.mux.Unlock()
	c.pathkey[syspath] = hash
	c.keypath[hash] = syspath
	return hash
}

// Instance of unlimited cache with hash-key<=>syspath pairs.
var hashcache = KeyThumbCache{
	keypath: map[string]string{},
	pathkey: map[string]string{},
}

// Produce string key for given hash bytes slice.
var keygen = base32.HexEncoding.WithPadding(base32.NoPadding)

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
			var syspath, ok = hashcache.Path(key.(string))
			if !ok {
				err = ErrNoHash
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
