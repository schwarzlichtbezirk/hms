package hms

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/base32"
	"io/ioutil"
	"os"
	"sync"

	"github.com/bluele/gcache"
	"gopkg.in/yaml.v3"
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

func loadhashcache(fpath string) {
	var err error

	var body []byte
	if body, err = ioutil.ReadFile(fpath); err == nil {
		if err = yaml.Unmarshal(body, &hashcache.keypath); err != nil {
			Log.Fatal("can not decode hashes cache: " + err.Error())
		}
	} else {
		Log.Println("can not read hashes cache: " + err.Error())
	}

	for key, path := range hashcache.keypath {
		hashcache.pathkey[path] = key
	}
}

const utf8bom = "\xef\xbb\xbf"

func savehashcache(fpath string) (err error) {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is MD5-hash of file system path encoded
# to base32, values are associated paths. Those keys
# used for files paths representations in URLs. You
# can modify keys to any alphanumerical text that
# should be unique.

`

	var file *os.File
	if file, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer file.Close()

	if _, err = file.WriteString(utf8bom); err != nil {
		return
	}
	if _, err = file.WriteString(intro); err != nil {
		return
	}

	var body []byte
	if body, err = yaml.Marshal(hashcache.keypath); err != nil {
		return
	}
	if _, err = file.Write(body); err != nil {
		return
	}
	return
}

// The End.
