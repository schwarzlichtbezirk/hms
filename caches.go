package hms

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/disintegration/gift"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/fsys"
)

// gcaches
var (
	// Files properties cache.
	// Key - system path, value - file property struct.
	propcache gcache.Cache

	// Public keys cache for authorization.
	pubkeycache gcache.Cache

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

// package caches
var (
	// cache with images thumbnails which are placed in box 256x256.
	thumbpkg *CachePackage
	// cache with images tiles, size of each tile is placed as sufix
	// of path in format "full/path/to/file.ext?144x108".
	tilespkg *CachePackage
)

// Error messages
var (
	ErrNoPUID      = errors.New("file with given puid not found")
	ErrUncacheable = errors.New("file format is uncacheable")
	ErrNotHD       = errors.New("image resolution does not fit to full HD")
	ErrNotDisk     = errors.New("file is not image of supported format")
)

// PathCache is unlimited cache with puid/fpath and fpath/puid values.
type PathCache struct {
	keypath map[Puid_t]string // puid/path key/values
	pathkey map[string]Puid_t // path/puid key/values
	mux     sync.RWMutex
}

// PUID returns cached PUID for specified system path.
func (c *PathCache) PUID(fpath string) (puid Puid_t, ok bool) {
	puid, ok = CatPathKey[fpath]
	if !ok {
		c.mux.RLock()
		puid, ok = c.pathkey[fpath]
		c.mux.RUnlock()
	}
	return
}

// Path returns cached system path of specified PUID (path unique identifier).
func (c *PathCache) Path(puid Puid_t) (fpath string, ok bool) {
	if puid < PUIDreserved {
		fpath, ok = CatKeyPath[puid]
	} else {
		c.mux.RLock()
		fpath, ok = c.keypath[puid]
		c.mux.RUnlock()
	}
	return
}

// MakePUID generates new path unique ID.
func (c *PathCache) MakePUID() (puid Puid_t) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	var n = 0
	for ok := true; puid < PUIDreserved || ok; _, ok = c.keypath[puid] {
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
func (c *PathCache) Cache(fpath string) (puid Puid_t) {
	var ok bool
	if puid, ok = c.PUID(fpath); ok {
		return
	}

	puid = c.MakePUID()

	c.mux.Lock()
	defer c.mux.Unlock()
	c.pathkey[fpath] = puid
	c.keypath[puid] = fpath
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

	var puid Puid_t
	if err := puid.Set(pref); err == nil {
		if fpath, ok := syspathcache.Path(puid); ok {
			if suff != "" { // prevent modify original path if suffix is absent
				fpath = path.Join(fpath, suff)
			}
			return fpath
		}
	}
	return shrpath
}

// Instance of unlimited cache with PUID<=>syspath pairs.
var syspathcache = PathCache{
	keypath: map[Puid_t]string{},
	pathkey: map[string]Puid_t{},
}

// DirCache is unlimited cache with puid/DirProp values.
type DirCache struct {
	keydir map[Puid_t]DirProp
	mux    sync.RWMutex
}

// Get value from cache.
func (c *DirCache) Get(puid Puid_t) (dp DirProp, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	dp, ok = c.keydir[puid]
	return
}

// Set value to cache.
func (c *DirCache) Set(puid Puid_t, dp DirProp) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.keydir[puid] = dp
}

// Category returns PUIDs list of directories where number
// of files of given category is more then given percent.
func (c *DirCache) Category(ctgr int, percent float64) (ret []Puid_t) {
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
func (c *DirCache) Categories(cats []int, percent float64) (ret []Puid_t) {
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
	keydir: map[Puid_t]DirProp{},
}

// GpsInfo describes GPS-data from the photos:
// latitude, longitude, altitude and creation time.
type GpsInfo struct {
	DateTime  unix_t  `json:"time" yaml:"time"` // photo creation date/time in Unix milliseconds
	Latitude  float64 `json:"lat" yaml:"lat"`
	Longitude float64 `json:"lon" yaml:"lon"`
	Altitude  float32 `json:"alt,omitempty" yaml:"alt,omitempty"`
}

// GpsCache inherits sync.Map and encapsulates functionality for cache.
type GpsCache struct {
	sync.Map
}

var gpscache GpsCache

// Prepares caches depends of previously loaded configuration.
func initcaches() {
	// init properties cache
	propcache = gcache.New(cfg.PropCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath = key.(string)
			if puid, ok := CatPathKey[syspath]; ok {
				var fk FileKit
				fk.NameVal = CatNames[puid]
				fk.TypeVal = FTctgr
				fk.PUIDVal = puid
				fk.SetTmb(MimeDis)
				ret = &fk
				return
			}
			var fi fs.FileInfo
			if fi, err = StatFile(syspath); err != nil {
				return
			}
			ret = MakeProp(syspath, fi)
			return
		}).
		Build()

	// init public keys cache
	pubkeycache = gcache.New(10).LRU().Expiration(15 * time.Second).Build()

	// init converted media files cache
	mediacache = gcache.New(cfg.MediaCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = syspathcache.Path(key.(Puid_t))
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
			case IsTypeNativeImg(ext):
				err = ErrUncacheable
				return // uncacheable type
			case IsTypeNonalpha(ext):
			case IsTypeAlpha(ext):
			default:
				err = ErrUncacheable
				return // uncacheable type
			}

			var file io.ReadCloser
			if file, err = OpenFile(syspath); err != nil {
				return // can not open file
			}
			defer file.Close()

			var ftype string
			var src image.Image
			if src, ftype, err = image.Decode(file); err != nil {
				if src == nil { // skip "short Huffman data" or others errors with partial results
					return // can not decode file by any codec
				}
			}

			ret, err = ToNativeImg(src, ftype)
			return
		}).
		Build()

	// init converted media files cache
	hdcache = gcache.New(cfg.MediaCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath, ok = syspathcache.Path(key.(Puid_t))
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

			var orientation = OrientNormal
			if ek, ok := prop.(*ExifKit); ok {
				var wdh, hgt int
				if ek.Width > ek.Height {
					wdh, hgt = cfg.HDResolution[0], cfg.HDResolution[1]
				} else {
					wdh, hgt = cfg.HDResolution[1], cfg.HDResolution[0]
				}
				if ek.Width <= wdh && ek.Height <= hgt {
					err = ErrNotHD
					return // does not fit to HD
				}
				orientation = ek.Orientation
			}

			var file io.ReadCloser
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

			var wdh, hgt int
			if src.Bounds().Dx() > src.Bounds().Dy() {
				wdh, hgt = cfg.HDResolution[0], cfg.HDResolution[1]
			} else {
				wdh, hgt = cfg.HDResolution[1], cfg.HDResolution[0]
			}

			if src.Bounds().In(image.Rect(0, 0, wdh, hgt)) {
				err = ErrNotHD
				return // does not fit to HD
			}

			var fltlst = AddOrientFilter([]gift.Filter{
				gift.ResizeToFit(wdh, hgt, gift.LinearResampling),
			}, orientation)
			var filter = gift.New(fltlst...)
			var img = image.NewRGBA(filter.Bounds(src.Bounds()))
			if img.Pix == nil {
				err = ErrImgNil
				return // out of memory
			}
			filter.Draw(img, src)
			dst = img

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

// MakeTile produces new tile object.
func MakeTile(r io.Reader, wdh, hgt int, orientation int) (md *MediaData, err error) {
	var ftype string
	var src, dst image.Image
	if src, ftype, err = image.Decode(r); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}

	switch orientation {
	case OrientCwHorzReversed, OrientCw, OrientAcwHorzReversed, OrientAcw:
		wdh, hgt = hgt, wdh
	}
	var fltlst = AddOrientFilter([]gift.Filter{
		gift.ResizeToFill(wdh, hgt, gift.LinearResampling, gift.CenterAnchor),
	}, orientation)
	var filter = gift.New(fltlst...)
	var img = image.NewRGBA(filter.Bounds(src.Bounds()))
	if img.Pix == nil {
		err = ErrImgNil
		return // out of memory
	}
	filter.Draw(img, src)
	dst = img

	return ToNativeImg(dst, ftype)
}

// GetCachedTile tries to extract existing tile from cache, otherwise
// makes new one and put it to cache.
func GetCachedTile(syspath string, wdh, hgt int) (md *MediaData, err error) {
	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)

	// try to extract tile from package
	if md, err = tilespkg.GetImage(tilepath); err != nil || md != nil {
		return
	}

	var r io.ReadCloser
	if r, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer r.Close()

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		return // can not get properties
	}
	var fp = prop.(Pather)
	if fp.Type() < 0 {
		err = ErrNotFile
		return
	}

	var orientation = OrientNormal
	if ek, ok := prop.(*ExifKit); ok {
		orientation = ek.Orientation
	}

	if md, err = MakeTile(r, wdh, hgt, orientation); err != nil {
		return
	}

	// push tile to package
	err = tilespkg.PutImage(tilepath, md)
	return
}

// CachePackage describes package cache functionality.
// Package splitted in two files - tags table file and
// cached images data file.
type CachePackage struct {
	*fsys.Package
	WPT wpk.WriteSeekCloser // package tags part
	WPF wpk.WriteSeekCloser // package files part
}

// InitCacheWriter opens existing cache with given file name placed in
// cache directory, or creates new cache file if no one found.
func InitCacheWriter(fname string) (cw *CachePackage, err error) {
	var pkgpath = wpk.MakeTagsPath(path.Join(CachePath, fname))
	var datpath = wpk.MakeDataPath(path.Join(CachePath, fname))
	cw = &CachePackage{}
	defer func() {
		if err != nil {
			if cw.WPT != nil {
				cw.WPT.Close()
				cw.WPT = nil
			}
			if cw.WPF != nil {
				cw.WPF.Close()
				cw.WPF = nil
			}
		}
	}()

	var ok, _ = PathExists(pkgpath)
	if cw.WPT, err = os.OpenFile(pkgpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if cw.WPF, err = os.OpenFile(datpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if ok {
		if cw.Package, err = fsys.OpenPackage(pkgpath); err != nil {
			return
		}
		var offset, size = cw.DataPos()
		if _, err = cw.WPF.Seek(int64(offset+size), io.SeekStart); err != nil {
			return
		}
	} else {
		cw.Package = fsys.NewPackage(datpath)
		if err = cw.Begin(cw.WPT); err != nil {
			return
		}
		cw.Package.Info().
			Put(wpk.TIDlabel, wpk.TagString(fname))
	}
	return
}

// Close saves actual tags table and closes opened cache.
func (cw *CachePackage) Close() (err error) {
	if et := cw.Sync(cw.WPT, cw.WPF); et != nil && err == nil {
		err = et
	}
	if et := cw.WPT.Close(); et != nil && err == nil {
		err = et
	}
	if et := cw.WPF.Close(); et != nil && err == nil {
		err = et
	}
	cw.WPT, cw.WPF = nil, nil
	return
}

// GetImage extracts image from the cache with given file name.
func (cw *CachePackage) GetImage(fpath string) (md *MediaData, err error) {
	if ts, ok := cw.Tagset(fpath); ok {
		var str string
		var mime Mime_t
		if str, ok = ts.String(wpk.TIDmime); !ok {
			return
		}
		if mime, ok = MimeVal[str]; !ok {
			return
		}

		var file io.ReadCloser
		if file, err = cw.OpenTagset(ts); err != nil {
			return
		}
		defer file.Close()

		var size = ts.Size()
		var buf = make([]byte, size)
		if _, err = file.Read(buf); err != nil {
			return
		}
		md = &MediaData{
			Data: buf,
			Mime: mime,
		}
	}
	return
}

// PutImage puts thumbnail to package.
func (cw *CachePackage) PutImage(fpath string, md *MediaData) (err error) {
	var ts *wpk.Tagset_t
	if ts, err = cw.PackData(cw.WPF, bytes.NewReader(md.Data), fpath); err != nil {
		return
	}
	ts.Put(wpk.TIDmime, wpk.TagString(MimeStr[md.Mime]))
	return
}

// PackInfo writes info to log about opened cache.
func PackInfo(fname string, pack wpk.Packager) {
	var num, size int64
	pack.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		num++
		return true
	})
	if ts, ok := pack.Tagset(""); ok {
		size = ts.Size()
	}
	Log.Infof("package '%s': cached %d files on %d bytes", fname, num, size)
}

func initpackages() (err error) {
	if thumbpkg, err = InitCacheWriter(tmbfile); err != nil {
		return
	}
	PackInfo(tmbfile, thumbpkg)

	if tilespkg, err = InitCacheWriter(tilfile); err != nil {
		return
	}
	PackInfo(tilfile, tilespkg)

	return nil
}

// The End.
