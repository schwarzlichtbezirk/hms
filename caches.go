package hms

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/disintegration/gift"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/fsys"
	"xorm.io/xorm"
	"xorm.io/xorm/names"

	_ "github.com/mattn/go-sqlite3"
)

// package caches
var (
	// cache with images thumbnails which are placed in box 256x256.
	thumbpkg *CachePackage
	// cache with images tiles, size of each tile is placed as sufix
	// of path in format "full/path/to/file.ext?144x108".
	tilespkg *CachePackage
)

var xormStorage *xorm.Engine

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
	var session = xormStorage.NewSession()
	defer session.Close()
	return f(session)
}

type TempCell[T any] struct {
	Data *T
	Wait *time.Timer
}

// PathStore sqlite3 item of unlimited cache with puid/syspath values.
type PathStore struct {
	Puid Puid_t `xorm:"pk autoincr"`
	Path string `xorm:"notnull unique index"`
}

// Store is struct of sqlite3 item for cache with puid/T values.
type Store[T any] struct {
	Puid Puid_t `xorm:"pk" json:"puid" yaml:"puid" xml:"puid,attr"`
	Prop T      `xorm:"extends" json:"prop" yaml:"prop" xml:"prop"`
}

type (
	DirStore  Store[DirProp]
	ExifStore Store[ExifProp]
	TagStore  Store[TagProp]
)

var (
	pathcache = NewBimap[Puid_t, string]()   // Bidirectional map for PUIDs and system paths.
	dircache  = NewCache[Puid_t, DirProp]()  // LRU cache for directories.
	exifcache = NewCache[Puid_t, ExifProp]() // FIFO cache for EXIF tags.
	tagcache  = NewCache[Puid_t, TagProp]()  // FIFO cache for ID3 tags.

	tmbcache  = NewCache[Puid_t, MediaData]() // FIFO cache with files embedded thumbnails.
	tilecache = NewCache[Puid_t, *TileProp]() // FIFO cache with set of available tiles.

	mediacache = NewCache[Puid_t, MediaData]() // FIFO cache with processed media files.
	hdcache    = NewCache[Puid_t, MediaData]() // FIFO cache with converted to HD resolution images.

	diskcache = NewCache[string, TempCell[DiskFS]]()     // LRU cache with temporary opened ISO disks.
	pubkcache = NewCache[[32]byte, TempCell[struct{}]]() // LRU cache with public keys.
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
	if ok = !ep.IsZero(); ok {
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
	if est.Prop.Latitude != 0 || est.Prop.Longitude != 0 {
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
	if ok = !tp.IsZero(); ok {
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
	DateTime  Time    `xorm:"DateTime" json:"time" yaml:"time" xml:"time,attr"` // photo creation date/time in Unix milliseconds
	Latitude  float64 `json:"lat" yaml:"lat" xml:"lat,attr"`
	Longitude float64 `json:"lon" yaml:"lon" xml:"lon,attr"`
	Altitude  float32 `json:"alt,omitempty" yaml:"alt,omitempty" xml:"alt,omitempty,attr"`
}

// FromProp fills fields with values from ExifProp.
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

// MediaCacheGet returns media file with given PUID converted to acceptable
// for browser format, with media cache usage.
func MediaCacheGet(session *Session, puid Puid_t) (md MediaData, err error) {
	var ok bool
	if md, ok = mediacache.Peek(puid); ok {
		return
	}

	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
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

	md, err = ToNativeImg(src, ftype)
	mediacache.Push(puid, md)
	mediacache.ToLimit(cfg.MediaCacheMaxNum)
	return
}

// HdCacheGet returns image with given PUID converted to HD resolution,
// with HD-images cache usage.
func HdCacheGet(session *Session, puid Puid_t) (md MediaData, err error) {
	var ok bool
	if md, ok = hdcache.Peek(puid); ok {
		return
	}

	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
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

	md, err = ToNativeImg(dst, ftype)
	hdcache.Push(puid, md)
	hdcache.ToLimit(cfg.HdCacheMaxNum)
	return
}

func DiskCacheGet(syspath string) (disk *DiskFS, err error) {
	var cell TempCell[DiskFS]
	var ok bool

	if cell, ok = diskcache.Get(syspath); ok {
		cell.Wait.Reset(cfg.DiskCacheExpire)
		disk = cell.Data
		return
	}
	var ext = GetFileExt(syspath)
	if !IsTypeISO(ext) {
		err = ErrNotDisk
		return
	}
	if disk, err = NewDiskFS(syspath); err != nil {
		return
	}

	cell.Data = disk
	cell.Wait = time.AfterFunc(cfg.DiskCacheExpire, func() {
		diskcache.Remove(syspath)
	})
	diskcache.Set(syspath, cell)
	return
}

// InitCaches prepares caches.
func InitCaches() {
	diskcache.OnRemove(func(syspath string, cell TempCell[DiskFS]) {
		cell.Wait.Stop()
		cell.Data.Close()
	})
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

// InitStorage inits database caches engine.
func InitStorage() (err error) {
	if xormStorage, err = xorm.NewEngine(xormDriverName, path.Join(CachePath, dirfile)); err != nil {
		return
	}
	xormStorage.SetMapper(names.GonicMapper{})
	var xlb = XormLoggerBridge{
		Logger: Log,
	}
	xlb.ShowSQL(devmode)
	xormStorage.SetLogger(&xlb)

	_, err = SqlSession(func(session *Session) (res any, err error) {
		if err = session.Sync(&PathStore{}, &DirStore{}, &ExifStore{}, &TagStore{}); err != nil {
			return
		}

		// fill path_store & file_store with predefined items
		var ok bool
		if ok, err = session.IsTableEmpty(&PathStore{}); err != nil {
			return
		}
		if ok {
			var ctgrpath = make([]PathStore, PUIDcache-1)
			for puid, path := range CatKeyPath {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
			}
			for puid := Puid_t(len(CatKeyPath) + 1); puid < PUIDcache; puid++ {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = fmt.Sprintf("<reserved%d>", puid)
			}
			if _, err = session.Insert(&ctgrpath); err != nil {
				return
			}
		}
		return
	})
	return
}

// LoadPathCache loads whole path table from database into cache.
func LoadPathCache() (err error) {
	var session = xormStorage.NewSession()
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
	var session = xormStorage.NewSession()
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
