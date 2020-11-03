package hms

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"image"
	"image/jpeg"
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
	// Key - path unique ID, value - thumbnail image in MediaData.
	thumbcache gcache.Cache

	// Converted media files cache.
	// Key - path unique ID, value - media file in MediaData.
	mediacache gcache.Cache
)

// Error messages
var (
	ErrNoPUID      = errors.New("file with given puid not found")
	ErrUncacheable = errors.New("file format is uncacheable")
)

// Unlimited cache with puid/syspath and syspath/puid values.
type KeyThumbCache struct {
	keypath map[string]string // puid/path key/values
	pathkey map[string]string // path/puid key/values
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
	var n = 0
	var buf [10]byte
	for ok = true; ok; _, ok = c.keypath[puid] {
		if n == 10 {
			switch {
			case cfg.PUIDsize < 3:
				cfg.PUIDsize = 3 // 16M pool
				n = 0
			case cfg.PUIDsize < 5:
				cfg.PUIDsize = 5 // 1T pool
				n = 0
			case cfg.PUIDsize < 10:
				cfg.PUIDsize = 10 // 10^24 pool
				n = 0
			}
		}
		if _, err := rand.Read(buf[:cfg.PUIDsize]); err != nil {
			panic(err)
		}
		puid = idenc.EncodeToString(buf[:cfg.PUIDsize])
		n++
	}

	c.pathkey[syspath] = puid
	c.keypath[puid] = syspath
	return puid
}

// Splits given share path to share prefix and remained suffix.
func SplitPrefSuff(shrpath string) (string, string) {
	for i, c := range shrpath {
		if c == '/' || c == '\\' {
			return shrpath[:i], shrpath[i+1:]
		} else if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return "", shrpath
		}
	}
	return shrpath, "" // root of share
}

// Brings any share path to system file path.
func UnfoldPath(shrpath string) string {
	var pref, suff = SplitPrefSuff(shrpath)
	if pref == "" {
		return shrpath
	}

	if path, ok := pathcache.Path(pref); ok {
		return path + suff
	}
	return shrpath
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
				for _, path := range CatPath {
					if path == syspath {
						var ck CatKit
						ck.Setup(path)
						ret, err = &ck, nil
						return
					}
				}
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

			var md *MediaData
			if md, err = FindTmb(prop, syspath); md != nil {
				prop.SetNTmb(TMB_cached)
				ret = md
			} else {
				prop.SetNTmb(TMB_reject)
			}
			return // ok
		}).
		Build()

	// init converted media files cache
	mediacache = gcache.New(cfg.ThumbCacheMaxNum).
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

			switch cp.(Proper).Type() {
			case FT_tga, FT_bmp, FT_tiff:
				var file *os.File
				if file, err = os.Open(syspath); err != nil {
					return // can not open file
				}
				defer file.Close()

				var img image.Image
				if img, _, err = image.Decode(file); err != nil {
					if img == nil { // skip "short Huffman data" or others errors with partial results
						return // can not decode file by any codec
					}
				}

				var buf bytes.Buffer
				if err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
					return // can not write jpeg
				}
				ret = &MediaData{
					Data: buf.Bytes(),
					Mime: "image/jpeg",
				}
				return

			case FT_dds, FT_psd:
				var file *os.File
				if file, err = os.Open(syspath); err != nil {
					return // can not open file
				}
				defer file.Close()

				var img image.Image
				if img, _, err = image.Decode(file); err != nil {
					if img == nil { // skip "short Huffman data" or others errors with partial results
						return // can not decode file by any codec
					}
				}

				var buf bytes.Buffer
				if err = thumbpngenc.Encode(&buf, img); err != nil {
					return // can not write png
				}
				ret = &MediaData{
					Data: buf.Bytes(),
					Mime: "image/png",
				}
				return
			}

			err = ErrUncacheable
			return // uncacheable type
		}).
		Build()
}

// The End.
