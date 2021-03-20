package hms

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
)

// gcaches
var (
	// Files properties cache.
	// Key - system path, value - file property struct.
	propcache gcache.Cache

	// Public keys cache for authorization.
	pubkeycache gcache.Cache

	// Thumbnails cache.
	// Key - path unique ID, value - thumbnail image in MediaData.
	thumbcache gcache.Cache

	// Converted media files cache.
	// Key - path unique ID, value - media file in MediaData.
	mediacache gcache.Cache

	// Opened disks cache.
	// Key - ISO image system path, value - disk data.
	diskcache gcache.Cache
)

// Produce base32 string representation of given random bytes slice.
var idenc = base32.HexEncoding.WithPadding(base32.NoPadding)

// Error messages
var (
	ErrNoPUID      = errors.New("file with given puid not found")
	ErrUncacheable = errors.New("file format is uncacheable")
	ErrNotDisk     = errors.New("file is not image of supported format")
)

// PathCache is unlimited cache with puid/syspath and syspath/puid values.
type PathCache struct {
	keypath map[string]string // puid/path key/values
	pathkey map[string]string // path/puid key/values
	mux     sync.RWMutex
}

// PUID returns cached PUID for specified system path.
func (c *PathCache) PUID(syspath string) (puid string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	puid, ok = c.pathkey[syspath]
	return
}

// Path returns cached system path of specified PUID (path unique identifier).
func (c *PathCache) Path(puid string) (syspath string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	syspath, ok = c.keypath[puid]
	return
}

// MakePUID generates new path unique ID.
func (c *PathCache) MakePUID() string {
	c.mux.RLock()
	defer c.mux.RUnlock()

	var puid string
	var n = 0
	var buf [10]byte
	for ok := true; ok; _, ok = c.keypath[puid] {
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
	return puid
}

// Cache returns cached PUID for specified system path, or make it and put into cache.
func (c *PathCache) Cache(syspath string) string {
	if puid, ok := c.PUID(syspath); ok {
		return puid
	}

	var puid = c.MakePUID()

	c.mux.Lock()
	defer c.mux.Unlock()
	c.pathkey[syspath] = puid
	c.keypath[puid] = syspath
	return puid
}

var puidsym = (func() (t [256]bool) {
	for i := '0'; i <= '9'; i++ {
		t[i] = true
	}
	for i := 'A'; i <= 'Z'; i++ {
		t[i] = true
	}
	for i := 'a'; i <= 'z'; i++ {
		t[i] = true
	}
	t['_'] = true
	t['-'] = true
	return
})()

// SplitPrefSuff splits given share path to share prefix and remained suffix.
func SplitPrefSuff(shrpath string) (string, string) {
	for i, c := range shrpath {
		if c == '/' || c == '\\' {
			return shrpath[:i], shrpath[i+1:]
		} else if !puidsym[c] {
			return "", shrpath
		}
	}
	return shrpath, "" // root of share
}

// UnfoldPath brings any share path to system file path.
func UnfoldPath(shrpath string) string {
	var pref, suff = SplitPrefSuff(shrpath)
	if pref == "" {
		return shrpath
	}

	if fpath, ok := pathcache.Path(pref); ok {
		return path.Join(fpath, suff)
	}
	return shrpath
}

// Instance of unlimited cache with PUID<=>syspath pairs.
var pathcache = PathCache{
	keypath: map[string]string{},
	pathkey: map[string]string{},
}

// DirCache is unlimited cache with puid/DirProp values.
type DirCache struct {
	keydir map[string]DirProp
	mux    sync.RWMutex
}

// Get value from cache.
func (c *DirCache) Get(puid string) (dp DirProp, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	dp, ok = c.keydir[puid]
	return
}

// Set value to cache.
func (c *DirCache) Set(puid string, dp DirProp) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.keydir[puid] = dp
}

// Category returns PUIDs list of directories where number
// of files of given category is more then given percent.
func (c *DirCache) Category(ctgr int, percent float64) (ret []string) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	for puid, dp := range c.keydir {
		var sum int
		for _, v := range dp.FGrp {
			sum += v
		}
		if sum > 0 && float64(dp.FGrp[ctgr])/float64(sum) > percent {
			ret = append(ret, puid)
		}
	}
	return
}

// Categories return PUIDs list of directories where number
// of files of any given categories is more then given percent.
func (c *DirCache) Categories(cats []int, percent float64) (ret []string) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	for puid, dp := range c.keydir {
		var sum int
		for _, v := range dp.FGrp {
			sum += v
		}
		var cs int
		for _, ci := range cats {
			cs += dp.FGrp[ci]
		}
		if sum > 0 && float64(cs)/float64(sum) > percent {
			ret = append(ret, puid)
		}
	}
	return
}

// Instance of unlimited cache with puid/DirProp values.
var dircache = DirCache{
	keydir: map[string]DirProp{},
}

// Prepares caches depends of previously loaded configuration.
func initcaches() {
	// init properties cache
	propcache = gcache.New(cfg.PropCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath = key.(string)
			var fi os.FileInfo
			if fi, err = os.Stat(syspath); err != nil {
				for _, fpath := range CatPath {
					if fpath == syspath {
						var ck CatKit
						ck.Setup(fpath)
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

	// init public keys cache
	pubkeycache = gcache.New(10).LRU().Expiration(15 * time.Second).Build()

	// init thumbnails cache
	thumbcache = gcache.New(cfg.ThumbCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = pathcache.Path(key.(string))
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var pv interface{}
			if pv, err = propcache.Get(syspath); err != nil {
				return // can not get properties
			}
			var prop = pv.(Pather)
			if prop.NTmb() == TMBreject {
				err = ErrNotThumb
				return // thumbnail rejected
			}

			var md *MediaData
			if md, err = FindTmb(prop, syspath); md != nil {
				prop.SetNTmb(TMBcached)
				ret = md
			} else {
				prop.SetNTmb(TMBreject)
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

			var prop interface{}
			if prop, err = propcache.Get(syspath); err != nil {
				return // can not get properties
			}
			var fp = prop.(Pather)
			if fp.Type() < 0 {
				err = ErrNotFile
				return
			}

			switch fp.Type() {
			case FTtga, FTbmp, FTtiff:
				var file io.ReadSeekCloser
				if file, err = OpenFile(syspath); err != nil {
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

			case FTdds, FTpsd:
				var file io.ReadSeekCloser
				if file, err = OpenFile(syspath); err != nil {
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

	diskcache = gcache.New(0).
		Simple().
		Expiration(cfg.DiskCacheExpire).
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var ext = strings.ToLower(path.Ext(key.(string)))
			if ext == ".iso" {
				return NewDiskISO(key.(string))
			}
			err = ErrNotDisk
			return
		}).
		EvictedFunc(func(key, value interface{}) {
			value.(io.Closer).Close()
		}).
		PurgeVisitorFunc(func(key, value interface{}) {
			value.(io.Closer).Close()
		}).
		Build()
}

// The End.
