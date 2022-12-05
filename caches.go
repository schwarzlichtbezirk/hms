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
	"github.com/mattn/go-sqlite3"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/fsys"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
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

const xormDriverName = "sqlite3"

var xormEngine *xorm.Engine

// Error messages
var (
	ErrNoPUID      = errors.New("file with given puid not found")
	ErrUncacheable = errors.New("file format is uncacheable")
	ErrNotHD       = errors.New("image resolution does not fit to full HD")
	ErrNotDisk     = errors.New("file is not image of supported format")
)

// ToSlash brings filenames to true slashes.
var ToSlash = wpk.ToSlash

// PathStarts check up that given file path has given parental path.
func PathStarts(fpath, prefix string) bool {
	if len(fpath) < len(prefix) {
		return false
	}
	if prefix == "" || prefix == "." || fpath == prefix {
		return true
	}
	if fpath[:len(prefix)] == prefix &&
		(prefix[len(prefix)-1] == '/' || fpath[len(prefix)] == '/') {
		return true
	}
	return false
}

// UpsertOne insert struct to database table,
// or update if some one already present with given identifier.
func UpsertOne(val interface{}) (err error) {
	switch xormDriverName {
	case "sqlite3":
		if _, err = xormEngine.InsertOne(val); err != nil {
			var serr sqlite3.Error
			if errors.As(err, &serr) && serr.ExtendedCode == 1555 {
				_, err = xormEngine.Update(val)
			}
		}
	default:
		if affected, _ := xormEngine.InsertOne(val); affected == 0 {
			_, err = xormEngine.Update(val)
		}
	}
	return
}

type PathInfo struct {
	Path string `xorm:"notnull unique index"`
	Type FT_t   `xorm:"default 0"`
	Size int64  `xorm:"default 0"`
	Time Unix_t `xorm:"default 0"`
}

// Setup sets size and time from file info.
func (pi *PathInfo) Setup(fi fs.FileInfo) {
	pi.Size = fi.Size()
	pi.Time = UnixJS(fi.ModTime())
}

// IsDiff returns true whether struct differs from file info.
func (pi *PathInfo) IsDiff(fi fs.FileInfo) bool {
	return pi.Size != fi.Size() || pi.Time != UnixJS(fi.ModTime())
}

// DirCacheItem sqlite3 item of unlimited cache with puid/syspath values.
type PathCacheItem struct {
	Puid     Puid_t `xorm:"pk autoincr"`
	PathInfo `xorm:"extends"`
}

func (c *PathCacheItem) TableName() string {
	return "path_cache"
}

// DirCacheItem sqlite3 item of unlimited cache with puid/FileGroup values.
type DirCacheItem struct {
	Puid    Puid_t `xorm:"pk"`
	DirProp `xorm:"extends"`
}

func (c *DirCacheItem) TableName() string {
	return "dir_cache"
}

type DirCacheKit struct {
	Puid     Puid_t `xorm:"pk"`
	PathInfo `xorm:"extends"`
	DirProp  `xorm:"extends"`
}

// DirCacheItem sqlite3 item of unlimited cache with puid/ExifProp values.
type ExifCacheItem struct {
	Puid     Puid_t `xorm:"pk"`
	ExifProp `xorm:"extends"`
}

func (c *ExifCacheItem) TableName() string {
	return "exif_cache"
}

// TagCacheItem sqlite3 item of unlimited cache with puid/TagProp values.
type TagCacheItem struct {
	Puid    Puid_t `xorm:"pk"`
	TagProp `xorm:"extends"`
}

func (c *TagCacheItem) TableName() string {
	return "tag_cache"
}

// PathCachePUID returns cached PUID for specified system path.
func PathCachePUID(fpath string) (puid Puid_t, ok bool) {
	var err error
	var pc PathCacheItem
	if ok, err = xormEngine.Cols("puid").Where("path=?", fpath).Get(&pc); err != nil {
		panic(err)
	}
	puid = pc.Puid
	return
}

// PathCachePath returns cached system path of specified PUID (path unique identifier).
func PathCachePath(puid Puid_t) (fpath string, ok bool) {
	var err error
	var pc PathCacheItem
	if ok, err = xormEngine.Cols("path").ID(puid).Get(&pc); err != nil {
		panic(err)
	}
	fpath = pc.Path
	return
}

// PathCacheSure returns cached PUID for specified system path, or make it and put into cache.
func PathCacheSure(fpath string) (puid Puid_t) {
	var err error
	var ok bool
	var pc PathCacheItem
	if ok, err = xormEngine.Cols("puid").Where("path=?", fpath).Get(&pc); err != nil {
		panic(err)
	}
	if ok {
		puid = pc.Puid
		return
	}

	pc.Path = fpath
	if _, err = xormEngine.InsertOne(&pc); err != nil {
		panic(err)
	}
	puid = pc.Puid
	return
}

// DirCacheGet returns value from directories cache.
func DirCacheGet(puid Puid_t) (dp DirProp, ok bool) {
	var err error
	var dc DirCacheItem
	if ok, err = xormEngine.ID(puid).Get(&dc); err != nil {
		panic(err)
	}
	dp = dc.DirProp
	return
}

// ExifCacheGet returns value from EXIF cache.
func ExifCacheGet(puid Puid_t) (ep ExifProp, ok bool) {
	var err error
	var eci ExifCacheItem
	if ok, err = xormEngine.ID(puid).Get(&eci); err != nil {
		panic(err)
	}
	ep = eci.ExifProp
	return
}

// ExifCacheSet puts value to EXIF cache.
func ExifCacheSet(eci *ExifCacheItem) (err error) {
	if eci.Latitude != 0 {
		var gi GpsInfo
		gi.FromProp(&eci.ExifProp)
		gpscache.Store(eci.Puid, gi)
	}
	if affected, _ := xormEngine.InsertOne(&eci); affected == 0 {
		_, err = xormEngine.ID(eci.Puid).AllCols().Omit("puid").Update(&eci)
	}
	return
}

// TagCacheGet returns value from tags cache.
func TagCacheGet(puid Puid_t) (tp TagProp, ok bool) {
	var err error
	var tc TagCacheItem
	if ok, err = xormEngine.ID(puid).Get(&tc); err != nil {
		panic(err)
	}
	tp = tc.TagProp
	return
}

// DirCacheCat returns PUIDs list of directories where number
// of files of given category is more then given percent.
func DirCacheCat(cat string, percent float64) (ret []Puid_t, err error) {
	const categoryCond = "(%s)/(other+video+audio+image+books+texts+packs+dir) > %f"
	err = xormEngine.Where(fmt.Sprintf(categoryCond, cat, percent)).Find(&ret)
	return
}

// GpsInfo describes GPS-data from the photos:
// latitude, longitude, altitude and creation time.
type GpsInfo struct {
	DateTime  Unix_t  `json:"time" yaml:"time" xml:"time,attr"` // photo creation date/time in Unix milliseconds
	Latitude  float64 `json:"lat" yaml:"lat" xml:"lat,attr"`
	Longitude float64 `json:"lon" yaml:"lon" xml:"lon,attr"`
	Altitude  float32 `json:"alt,omitempty" yaml:"alt,omitempty" xml:"alt,omitempty,attr"`
}

func (gi *GpsInfo) FromProp(ep *ExifProp) {
	gi.DateTime = ep.DateTime
	gi.Latitude = ep.Latitude
	gi.Longitude = ep.Longitude
	gi.Altitude = ep.Altitude
}

// GpsCache inherits sync.Map and encapsulates functionality for cache.
type GpsCache struct {
	sync.Map
}

// Count returns number of entries in the map.
func (gc *GpsCache) Count() (n int) {
	gc.Map.Range(func(key interface{}, value interface{}) bool {
		n++
		return true
	})
	return
}

// Range calls given closure for each GpsInfo in the map.
func (gc *GpsCache) Range(f func(Puid_t, *GpsInfo) bool) {
	gc.Map.Range(func(key, value interface{}) bool {
		return f(key.(Puid_t), value.(*GpsInfo))
	})
}

// Load gets all items with GPS information from EXIF stirage.
func (gc *GpsCache) Load() (err error) {
	const limit = 256
	var offset int
	for {
		var chunk []ExifCacheItem
		if err = xormEngine.Where("latitude != 0").Cols("datetime", "latitude", "longitude", "altitude").Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ec := range chunk {
			var gi GpsInfo
			gi.FromProp(&ec.ExifProp)
			gc.Store(ec.Puid, gi)
		}
		if limit > len(chunk) {
			break
		}
	}
	return
}

var gpscache GpsCache

// InitCaches prepares caches depends of previously loaded configuration.
func InitCaches() {
	// init properties cache
	propcache = gcache.New(cfg.PropCacheMaxNum).
		LRU().
		LoaderFunc(func(key interface{}) (ret interface{}, err error) {
			var syspath = key.(string)
			if puid, ok := CatPathKey[syspath]; ok {
				var ck CatKit
				ck.Setup(puid)
				ret = &ck
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
			var syspath, ok = PathCachePath(key.(Puid_t))
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var prop interface{}
			if prop, err = propcache.Get(syspath); err != nil {
				return // can not get properties
			}
			var fp = prop.(Pather)
			if fp.Type() != FTfile {
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
			var syspath, ok = PathCachePath(key.(Puid_t))
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var prop interface{}
			if prop, err = propcache.Get(syspath); err != nil {
				return // can not get properties
			}
			var fp = prop.(Pather)
			if fp.Type() != FTfile {
				err = ErrNotFile
				return
			}

			var orientation = OrientNormal
			if ek, ok := prop.(*ExifKit); ok {
				orientation = ek.Orientation
				if ek.Width > 0 && ek.Height > 0 {
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
				}
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

const (
	tidsz  = 2
	tagsz  = 2
	tssize = 2
)

// CachePackage describes package cache functionality.
// Package splitted in two files - tags table file and
// cached images data file.
type CachePackage struct {
	wpk.Package
	wpt wpk.WriteSeekCloser // package tags part
	wpf wpk.WriteSeekCloser // package files part
}

// InitCacheWriter opens existing cache with given file name placed in
// cache directory, or creates new cache file if no one found.
func InitCacheWriter(fname string) (cw *CachePackage, err error) {
	var pkgpath = wpk.MakeTagsPath(path.Join(CachePath, fname))
	var datpath = wpk.MakeDataPath(path.Join(CachePath, fname))
	cw = &CachePackage{
		Package: wpk.Package{
			FTT:       &wpk.FTT{},
			Workspace: ".",
		},
	}
	defer func() {
		if err != nil {
			if cw.wpt != nil {
				cw.wpt.Close()
				cw.wpt = nil
			}
			if cw.wpf != nil {
				cw.wpf.Close()
				cw.wpf = nil
			}
		}
	}()

	var ok, _ = PathExists(pkgpath)
	if cw.wpt, err = os.OpenFile(pkgpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if cw.wpf, err = os.OpenFile(datpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if ok {
		var r io.ReadSeekCloser
		if r, err = os.Open(pkgpath); err != nil {
			return
		}
		defer r.Close()

		if err = cw.ReadFTT(r); err != nil {
			return
		}

		if err = cw.Append(cw.wpt, cw.wpf); err != nil {
			return
		}
	} else {
		cw.Init(wpk.TypeSize{
			tidsz, tagsz, tssize,
		})

		if err = cw.Begin(cw.wpt, cw.wpf); err != nil {
			return
		}
		cw.Package.SetInfo().
			Put(wpk.TIDlabel, wpk.StrTag(fname))
	}
	if cw.Tagger, err = fsys.MakeTagger(datpath); err != nil {
		return
	}
	return
}

// Sync writes actual file tags table and true signature with settings.
func (cw *CachePackage) Sync() error {
	return cw.Package.Sync(cw.wpt, cw.wpf)
}

// Close saves actual tags table and closes opened cache.
func (cw *CachePackage) Close() (err error) {
	if et := cw.Sync(); et != nil && err == nil {
		err = et
	}
	if et := cw.wpt.Close(); et != nil && err == nil {
		err = et
	}
	if et := cw.wpf.Close(); et != nil && err == nil {
		err = et
	}
	cw.wpt, cw.wpf = nil, nil
	return
}

// GetImage extracts image from the cache with given file name.
func (cw *CachePackage) GetImage(fpath string) (md *MediaData, err error) {
	if ts, ok := cw.Tagset(fpath); ok {
		var str string
		var mime Mime_t
		if str, ok = ts.TagStr(wpk.TIDmime); !ok {
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
	var ts *wpk.TagsetRaw
	if ts, err = cw.PackData(cw.wpf, bytes.NewReader(md.Data), fpath); err != nil {
		return
	}
	var now = time.Now()
	ts.Put(wpk.TIDmtime, wpk.UnixTag(now))
	ts.Put(wpk.TIDatime, wpk.UnixTag(now))
	ts.Put(wpk.TIDmime, wpk.StrTag(MimeStr[md.Mime]))
	return
}

// PackInfo writes info to log about opened cache.
func PackInfo(fname string, pkg *wpk.Package) {
	var num, size int64
	pkg.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		num++
		return true
	})
	if ts, ok := pkg.Tagset(wpk.InfoName); ok {
		size = ts.Size()
	}
	Log.Infof("package '%s': cached %d files on %d bytes", fname, num, size)
}

// InitPackages opens all existing caches.
func InitPackages() (err error) {
	if thumbpkg, err = InitCacheWriter(tmbfile); err != nil {
		err = fmt.Errorf("inits thumbnails database: %w", err)
		return
	}
	PackInfo(tmbfile, &thumbpkg.Package)

	if tilespkg, err = InitCacheWriter(tilfile); err != nil {
		err = fmt.Errorf("inits tiles database: %w", err)
		return
	}
	PackInfo(tilfile, &tilespkg.Package)

	return nil
}

// InitXorm inits database caches engines.
func InitXorm() (err error) {
	if xormEngine, err = xorm.NewEngine(xormDriverName, path.Join(CachePath, dirfile)); err != nil {
		return
	}
	xormEngine.ShowSQL(true)
	xormEngine.SetMapper(names.GonicMapper{})
	if err = xormEngine.Sync(&PathCacheItem{}, &DirCacheItem{}, &ExifCacheItem{}, &TagCacheItem{}); err != nil {
		return
	}

	// fill path_cache with predefined items
	var ok bool
	if ok, err = xormEngine.IsTableEmpty(&PathCacheItem{}); err != nil {
		return
	}
	if ok {
		var ctgr = make([]PathCacheItem, PUIDcache-1)
		for puid, path := range CatKeyPath {
			ctgr[puid-1].Puid = puid
			ctgr[puid-1].PathInfo = PathInfo{
				Path: path,
				Type: FTctgr,
			}
		}
		for puid := Puid_t(len(CatKeyPath) + 1); puid < PUIDcache; puid++ {
			ctgr[puid-1].Puid = puid
			ctgr[puid-1].PathInfo = PathInfo{
				Path: fmt.Sprintf("<reserved%d>", puid),
				Type: FTctgr,
			}
		}
		if _, err = xormEngine.Insert(&ctgr); err != nil {
			return
		}
	}
	return
}

// The End.
