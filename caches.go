package hms

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/disintegration/gift"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/fsys"
	"xorm.io/xorm"
	"xorm.io/xorm/names"

	_ "github.com/mattn/go-sqlite3"
)

// gcaches
var (
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

type Session = xorm.Session

// SqlSession execute sql wrapped in a single session.
func SqlSession(f func(*Session) (any, error)) (any, error) {
	var session = xormEngine.NewSession()
	defer session.Close()
	return f(session)
}

// PathStore sqlite3 item of unlimited cache with puid/syspath values.
type PathStore struct {
	Puid Puid_t `xorm:"pk autoincr"`
	Path string `xorm:"notnull unique index"`
}

// Store is struct of sqlite3 item for cache with puid/T values.
type Store[T any] struct {
	Puid Puid_t `xorm:"pk"`
	Prop T      `xorm:"extends"`
}

type (
	FileStore Store[FileProp]
	DirStore  Store[DirProp]
	ExifStore Store[ExifProp]
	TagStore  Store[TagProp]
)

var (
	pathcache = NewBimap[Puid_t, string]()   // Bidirectional map for PUIDs and system paths.
	filecache = NewCache[Puid_t, FileProp]() // LRU cache for files.
	dircache  = NewCache[Puid_t, DirProp]()  // LRU cache for directories.
	exifcache = NewCache[Puid_t, ExifProp]() // FIFO cache for EXIF tags.
	tagcache  = NewCache[Puid_t, TagProp]()  // FIFO cache for ID3 tags.

	tmbcache  = NewCache[Puid_t, MediaData]() // FIFO cache with files embedded thumbnails.
	tilecache = NewCache[Puid_t, TileProp]()  // FIFO cache with set of available tiles.
)

// PathStorePUID returns cached PUID for specified system path.
func PathStorePUID(session *Session, fpath string) (puid Puid_t, ok bool) {
	// try to get from memory cache
	if puid, ok = pathcache.GetRev(fpath); ok {
		return
	}
	// try to get from database
	var val uint64
	if ok, _ = session.Table("path_store").Cols("puid").Where("path=?", fpath).Get(&val); ok { // skip errors
		puid = Puid_t(val)
		pathcache.Set(puid, fpath) // update cache
		return
	}
	return
}

// PathStorePath returns cached system path of specified PUID (path unique identifier).
func PathStorePath(session *Session, puid Puid_t) (fpath string, ok bool) {
	// try to get from memory cache
	if fpath, ok = pathcache.GetDir(puid); ok {
		return
	}
	// try to get from database
	if ok, _ = session.Table("path_store").Cols("path").Where("puid=?", puid).Get(&fpath); ok { // skip errors
		pathcache.Set(puid, fpath) // update cache
		return
	}
	return
}

// PathStoreCache returns cached PUID for specified system path, or make it and put into cache.
func PathStoreCache(session *Session, fpath string) (puid Puid_t) {
	var ok bool
	if puid, ok = PathStorePUID(session, fpath); ok {
		return
	}

	// set to database
	var pst = PathStore{
		Path: fpath,
	}
	if _, err := session.InsertOne(&pst); err != nil {
		panic(err)
	}
	puid = pst.Puid
	// set to memory cache
	pathcache.Set(puid, fpath)
	return
}

// FileStoreGet returns value from files properties cache.
func FileStoreGet(session *Session, puid Puid_t) (fp FileProp, ok bool) {
	// try to get from memory cache
	if fp, ok = filecache.Get(puid); ok {
		return
	}
	// try to get from database
	var fst FileStore
	if ok, _ = session.ID(puid).Get(&fst); ok { // skip errors
		fp = fst.Prop
		filecache.Set(puid, fp) // update cache
		return
	}
	return
}

// FileStoreSet puts value to files properties cache.
func FileStoreSet(session *Session, fst *FileStore) (err error) {
	// set to memory cache
	filecache.Set(fst.Puid, fst.Prop)
	// set to database
	if affected, _ := session.InsertOne(fst); affected == 0 {
		_, err = session.ID(fst.Puid).AllCols().Omit("puid").Update(fst)
	}
	return
}

// DirStoreGet returns value from directories cache.
func DirStoreGet(session *Session, puid Puid_t) (dp DirProp, ok bool) {
	// try to get from memory cache
	if dp, ok = dircache.Get(puid); ok {
		return
	}
	// try to get from database
	var dst DirStore
	if ok, _ = session.ID(puid).Get(&dst); ok { // skip errors
		dp = dst.Prop
		dircache.Set(puid, dp) // update cache
		return
	}
	return
}

// DirStoreSet puts value to directories cache.
func DirStoreSet(session *Session, dst *DirStore) (err error) {
	// set to memory cache
	dircache.Set(dst.Puid, dst.Prop)
	// set to database
	if affected, _ := session.InsertOne(dst); affected == 0 {
		_, err = session.ID(dst.Puid).AllCols().Omit("puid").Update(dst)
	}
	return
}

// ExifStoreGet returns value from EXIF cache.
func ExifStoreGet(session *Session, puid Puid_t) (ep ExifProp, ok bool) {
	// try to get from memory cache
	if ep, ok = exifcache.Peek(puid); ok {
		return
	}
	// try to get from database
	var est ExifStore
	if ok, _ = session.ID(puid).Get(&est); ok { // skip errors
		ep = est.Prop
		exifcache.Push(puid, ep) // update cache
		return
	}
	// try to extract from file
	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
		return
	}
	if err := ep.Extract(syspath); err != nil {
		ok = false
		return
	}
	if !ep.IsZero() {
		ExifStoreSet(session, &ExifStore{ // update database
			Puid: puid,
			Prop: ep,
		})
	}
	return
}

// ExifStoreSet puts value to EXIF cache.
func ExifStoreSet(session *Session, est *ExifStore) (err error) {
	// set to GPS cache
	if est.Prop.Latitude != 0 {
		var gi GpsInfo
		gi.FromProp(&est.Prop)
		gpscache.Store(est.Puid, gi)
	}
	// set to memory cache
	exifcache.Push(est.Puid, est.Prop)
	// set to database
	if affected, _ := session.InsertOne(est); affected == 0 {
		_, err = session.ID(est.Puid).AllCols().Omit("puid").Update(est)
	}
	return
}

// TagStoreGet returns value from tags cache.
func TagStoreGet(session *Session, puid Puid_t) (tp TagProp, ok bool) {
	// try to get from memory cache
	if tp, ok = tagcache.Peek(puid); ok {
		return
	}
	// try to get from database
	var tst TagStore
	if ok, _ = session.ID(puid).Get(&tst); ok { // skip errors
		tp = tst.Prop
		tagcache.Push(puid, tp) // update cache
		return
	}
	// try to extract from file
	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
		return
	}
	if err := tp.Extract(syspath); err != nil {
		ok = false
		return
	}
	if !tp.IsZero() {
		TagStoreSet(session, &TagStore{ // update database
			Puid: puid,
			Prop: tp,
		})
	}
	return
}

// TagStoreSet puts value to tags cache.
func TagStoreSet(session *Session, tst *TagStore) (err error) {
	// set to memory cache
	tagcache.Push(tst.Puid, tst.Prop)
	// set to database
	if affected, _ := session.InsertOne(tst); affected == 0 {
		_, err = session.ID(tst.Puid).AllCols().Omit("puid").Update(tst)
	}
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
	gc.Map.Range(func(key any, value any) bool {
		n++
		return true
	})
	return
}

// Range calls given closure for each GpsInfo in the map.
func (gc *GpsCache) Range(f func(Puid_t, GpsInfo) bool) {
	gc.Map.Range(func(key, value any) bool {
		var puid, okk = key.(Puid_t)
		var gps, okv = value.(GpsInfo)
		if okk && okv {
			return f(puid, gps)
		} else {
			return true
		}
	})
}

var gpscache GpsCache

// InitCaches prepares caches depends of previously loaded configuration.
func InitCaches() {
	// init public keys cache
	pubkeycache = gcache.New(10).LRU().Expiration(15 * time.Second).Build()

	// init converted media files cache
	mediacache = gcache.New(cfg.MediaCacheMaxNum).
		LRU().
		LoaderFunc(func(key any) (ret any, err error) {
			var syspath, ok = pathcache.GetDir(key.(Puid_t))
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var ext = GetFileExt(syspath)
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
		LoaderFunc(func(key any) (ret any, err error) {
			var session = xormEngine.NewSession()
			defer session.Close()

			var puid = key.(Puid_t)
			var syspath, ok = pathcache.GetDir(puid)
			if !ok {
				err = ErrNoPUID
				return // file path not found
			}

			var orientation = OrientNormal
			if ep, ok := ExifStoreGet(session, puid); ok { // skip non-EXIF properties
				if ep.Orientation > 0 {
					orientation = ep.Orientation
				}
				if ep.Width > 0 && ep.Height > 0 {
					var wdh, hgt int
					if ep.Width > ep.Height {
						wdh, hgt = cfg.HDResolution[0], cfg.HDResolution[1]
					} else {
						wdh, hgt = cfg.HDResolution[1], cfg.HDResolution[0]
					}
					if ep.Width <= wdh && ep.Height <= hgt {
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
		LoaderFunc(func(key any) (ret any, err error) {
			var ext = strings.ToLower(path.Ext(key.(string)))
			if ext == ".iso" {
				return NewDiskISO(key.(string))
			}
			err = ErrNotDisk
			return
		}).
		EvictedFunc(func(_, value any) {
			value.(io.Closer).Close()
		}).
		PurgeVisitorFunc(func(_, value any) {
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
func (cw *CachePackage) GetImage(fpath string) (md MediaData, err error) {
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
		md.Data = buf
		md.Mime = mime
	}
	return
}

// PutImage puts thumbnail to package.
func (cw *CachePackage) PutImage(fpath string, md MediaData) (err error) {
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

	_, err = SqlSession(func(session *Session) (res any, err error) {
		if err = session.Sync(&PathStore{}, &FileStore{}, &DirStore{}, &ExifStore{}, &TagStore{}); err != nil {
			return
		}

		// fill path_store & file_store with predefined items
		var ok bool
		if ok, err = session.IsTableEmpty(&PathStore{}); err != nil {
			return
		}
		if ok {
			var tinit = UnixJSNow()
			var ctgrpath = make([]PathStore, PUIDcache-1)
			var ctgrfile = make([]FileStore, PUIDcache-1)
			for puid, path := range CatKeyPath {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				ctgrfile[puid-1].Puid = puid
				ctgrfile[puid-1].Prop = FileProp{
					Name: CatNames[puid],
					Type: FTctgr,
					Time: tinit,
				}
			}
			for puid := Puid_t(len(CatKeyPath) + 1); puid < PUIDcache; puid++ {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = fmt.Sprintf("<reserved%d>", puid)
				ctgrfile[puid-1].Puid = puid
				ctgrfile[puid-1].Prop = FileProp{
					Name: fmt.Sprintf("reserved #%d", puid),
					Type: FTctgr,
					Time: tinit,
				}
			}
			if _, err = session.Insert(&ctgrpath); err != nil {
				return
			}
			if _, err = session.Insert(&ctgrfile); err != nil {
				return
			}
		}
		return
	})
	return
}

// LoadPathCache loads whole path table from database into cache.
func LoadPathCache() (err error) {
	var session = xormEngine.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var chunk []PathStore
		if err = session.Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ps := range chunk {
			pathcache.Set(ps.Puid, ps.Path)
		}
		if limit > len(chunk) {
			break
		}
	}
	return
}

// LoadGpsCache loads all items with GPS information from EXIF table of storage into cache.
func LoadGpsCache() (err error) {
	var session = xormEngine.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var chunk []ExifStore
		if err = session.Where("latitude != 0").Cols("puid", "datetime", "latitude", "longitude", "altitude").Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ec := range chunk {
			var gi GpsInfo
			gi.FromProp(&ec.Prop)
			gpscache.Store(ec.Puid, gi)
		}
		if limit > len(chunk) {
			break
		}
	}
	return
}

// The End.
