package hms

import (
	"bytes"
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

	// Photos compressed to HD resolution.
	// Key - path unique ID, value - media file in MediaData.
	hdcache gcache.Cache

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
	ErrNotHD       = errors.New("image resolution does not fit to full HD")
	ErrNotDisk     = errors.New("file is not image of supported format")
)

// PathCache is unlimited cache with puid/syspath and syspath/puid values.
type PathCache struct {
	keypath map[PuidType]string // puid/path key/values
	pathkey map[string]PuidType // path/puid key/values
	mux     sync.RWMutex
}

// PUID returns cached PUID for specified system path.
func (c *PathCache) PUID(syspath string) (puid PuidType, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	puid, ok = c.pathkey[syspath]
	return
}

// Path returns cached system path of specified PUID (path unique identifier).
func (c *PathCache) Path(puid PuidType) (syspath string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	syspath, ok = c.keypath[puid]
	return
}

// MakePUID generates new path unique ID.
func (c *PathCache) MakePUID() (puid PuidType) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	var n = 0
	for ok := true; ok; _, ok = c.keypath[puid] {
		if n == 10 {
			cfg.PUIDlen++
			if cfg.PUIDlen > 12 {
				panic("PUID pool is exhausted")
			}
			n = 0
		}
		puid.Rand(cfg.PUIDlen * 5)
		n++
	}
	return
}

// Cache returns cached PUID for specified system path, or make it and put into cache.
func (c *PathCache) Cache(syspath string) (puid PuidType) {
	var ok bool
	if puid, ok = c.PUID(syspath); ok {
		return
	}

	puid = c.MakePUID()

	c.mux.Lock()
	defer c.mux.Unlock()
	c.pathkey[syspath] = puid
	c.keypath[puid] = syspath
	return
}

var puidsym = (func() (t [256]bool) {
	const encodeHex = "0123456789ABCDEFGHIJKLMNOPQRSTUV"
	for _, c := range encodeHex {
		t[c] = true
	}
	return
})()

// SplitPrefSuff splits given share path to share prefix and remained suffix.
func SplitPrefSuff(shrpath string) (string, string) {
	for i, c := range shrpath {
		if c == '/' || c == '\\' {
			return shrpath[:i], shrpath[i+1:]
		} else if int(c) >= len(puidsym) || !puidsym[c] {
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

	var puid PuidType
	if err := puid.Set(pref); err == nil {
		if fpath, ok := pathcache.Path(puid); ok {
			if suff != "" { // prevent modify original path if suffix is absent
				fpath = path.Join(fpath, suff)
			}
			return fpath
		}
	}
	return shrpath
}

// Instance of unlimited cache with PUID<=>syspath pairs.
var pathcache = PathCache{
	keypath: map[PuidType]string{},
	pathkey: map[string]PuidType{},
}

// DirCache is unlimited cache with puid/DirProp values.
type DirCache struct {
	keydir map[PuidType]DirProp
	mux    sync.RWMutex
}

// Get value from cache.
func (c *DirCache) Get(puid PuidType) (dp DirProp, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	dp, ok = c.keydir[puid]
	return
}

// Set value to cache.
func (c *DirCache) Set(puid PuidType, dp DirProp) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.keydir[puid] = dp
}

// Category returns PUIDs list of directories where number
// of files of given category is more then given percent.
func (c *DirCache) Category(ctgr int, percent float64) (ret []PuidType) {
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
func (c *DirCache) Categories(cats []int, percent float64) (ret []PuidType) {
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
	keydir: map[PuidType]DirProp{},
}

// Prepares caches depends of previously loaded configuration.
func initcaches() {
	// init properties cache
	propcache = gcache.New(cfg.PropCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath = key.(string)
			var fi os.FileInfo
			if fi, err = StatFile(syspath); err != nil {
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
			var syspath, ok = pathcache.Path(key.(PuidType))
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
				prop.SetTmb(TMBcached, md.Mime)
				ret = md
			} else {
				prop.SetTmb(TMBreject, "")
			}
			return // ok
		}).
		Build()

	// init converted media files cache
	mediacache = gcache.New(cfg.MediaCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = pathcache.Path(key.(PuidType))
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

			var ext = GetFileExt(fp.Name())
			switch {
			case IsTypeNonalpha(ext):
				var file VFile
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

			case IsTypeAlpha(ext):
				var file VFile
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

	// init converted media files cache
	hdcache = gcache.New(cfg.MediaCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = pathcache.Path(key.(PuidType))
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

			if ek, ok := prop.(ExifKit); ok {
				if (ek.Width <= 1920 && ek.Height <= 1080) ||
					(ek.Width <= 1080 && ek.Height <= 1920) {
					err = ErrNotHD
					return // does not fit to HD
				}
			}

			var file VFile
			if file, err = OpenFile(syspath); err != nil {
				return // can not open file
			}
			defer file.Close()

			var ftype string
			var src, dst image.Image
			if src, ftype, err = image.Decode(file); err != nil {
				if src == nil { // skip "short Huffman data" or others errors with partial results
					return // can not decode file by any codec
				}
			}
			if src.Bounds().In(hdhrect) || src.Bounds().In(hdvrect) {
				err = ErrNotHD
				return // does not fit to HD
			} else if src.Bounds().Dx() > src.Bounds().Dy() {
				var img = image.NewRGBA(hdhfilter.Bounds(src.Bounds()))
				hdhfilter.Draw(img, src)
				dst = img
			} else {
				var img = image.NewRGBA(hdvfilter.Bounds(src.Bounds()))
				hdvfilter.Draw(img, src)
				dst = img
			}

			return ToNativeImg(dst, ftype)
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
		EvictedFunc(func(_, value interface{}) {
			value.(io.Closer).Close()
		}).
		PurgeVisitorFunc(func(_, value interface{}) {
			value.(io.Closer).Close()
		}).
		Build()
}

// The End.
