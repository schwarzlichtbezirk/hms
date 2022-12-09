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
	"xorm.io/xorm"
	"xorm.io/xorm/names"

	_ "github.com/mattn/go-sqlite3"
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

type Session = xorm.Session

// SqlSession execute sql wrapped in a single session.
func SqlSession(f func(*Session) (interface{}, error)) (interface{}, error) {
	var session = xormEngine.NewSession()
	defer session.Close()
	return f(session)
}

type PathStore struct {
	Puid Puid_t `xorm:"pk autoincr"`
	Path string `xorm:"notnull unique index"`
}

// PathStore sqlite3 item of unlimited cache with puid/syspath values.
type FileStore struct {
	Puid     Puid_t `xorm:"pk"`
	FileProp `xorm:"extends"`
}

// DirStore sqlite3 item of unlimited cache with puid/FileGroup values.
type DirStore struct {
	Puid    Puid_t `xorm:"pk"`
	DirProp `xorm:"extends"`
}

// ExifStore sqlite3 item of unlimited cache with puid/ExifProp values.
type ExifStore struct {
	Puid     Puid_t `xorm:"pk"`
	ExifProp `xorm:"extends"`
}

// TagStore sqlite3 item of unlimited cache with puid/TagProp values.
type TagStore struct {
	Puid    Puid_t `xorm:"pk"`
	TagProp `xorm:"extends"`
}

var (
	pathcache = NewBimap[Puid_t, string]()
	filecache = NewCache[Puid_t, FileProp](0)
	dircache  = NewCache[Puid_t, DirProp](0)
	exifcache = NewCache[Puid_t, ExifProp](0)
	tagcache  = NewCache[Puid_t, TagProp](0)
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
	var ps = PathStore{
		Path: fpath,
	}
	if _, err := session.InsertOne(&ps); err != nil {
		panic(err)
	}
	puid = ps.Puid
	// set to memory cache
	pathcache.Set(puid, fpath)
	return
}

// DirStoreGet returns value from files properties cache.
func FileStoreGet(session *Session, puid Puid_t) (fp FileProp, ok bool) {
	// try to get from memory cache
	if fp, ok = filecache.Get(puid); ok {
		return
	}
	// try to get from database
	var fs FileStore
	if ok, _ = session.ID(puid).Get(&fs); ok { // skip errors
		fp = fs.FileProp
		filecache.Set(puid, fp) // update cache
	}
	return
}

// FileStoreSet puts value to files properties cache.
func FileStoreSet(session *Session, fs *FileStore) (err error) {
	// set to memory cache
	filecache.Set(fs.Puid, fs.FileProp)
	// set to database
	if affected, _ := session.InsertOne(fs); affected == 0 {
		_, err = session.ID(fs.Puid).AllCols().Omit("puid").Update(fs)
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
	var ds DirStore
	if ok, _ = session.ID(puid).Get(&ds); ok { // skip errors
		dp = ds.DirProp
		dircache.Set(puid, dp) // update cache
	}
	return
}

// DirStoreSet puts value to directories cache.
func DirStoreSet(session *Session, ds *DirStore) (err error) {
	// set to memory cache
	dircache.Set(ds.Puid, ds.DirProp)
	// set to database
	if affected, _ := session.InsertOne(ds); affected == 0 {
		_, err = session.ID(ds.Puid).AllCols().Omit("puid").Update(ds)
	}
	return
}

// ExifStoreGet returns value from EXIF cache.
func ExifStoreGet(session *Session, puid Puid_t) (ep ExifProp, ok bool) {
	// try to get from memory cache
	if ep, ok = exifcache.Get(puid); ok {
		return
	}
	// try to get from database
	var es ExifStore
	if ok, _ = session.ID(puid).Get(&es); ok { // skip errors
		ep = es.ExifProp
		exifcache.Set(puid, ep) // update cache
	}
	return
}

// ExifStoreSet puts value to EXIF cache.
func ExifStoreSet(session *Session, es *ExifStore) (err error) {
	// set to GPS cache
	if es.Latitude != 0 {
		var gi GpsInfo
		gi.FromProp(&es.ExifProp)
		gpscache.Store(es.Puid, gi)
	}
	// set to memory cache
	exifcache.Set(es.Puid, es.ExifProp)
	// set to database
	if affected, _ := session.InsertOne(es); affected == 0 {
		_, err = session.ID(es.Puid).AllCols().Omit("puid").Update(es)
	}
	return
}

// TagStoreGet returns value from tags cache.
func TagStoreGet(session *Session, puid Puid_t) (tp TagProp, ok bool) {
	// try to get from memory cache
	if tp, ok = tagcache.Get(puid); ok {
		return
	}
	// try to get from database
	var ts TagStore
	if ok, _ = session.ID(puid).Get(&ts); ok { // skip errors
		tp = ts.TagProp
		tagcache.Set(puid, tp) // update cache
	}
	return
}

// TagStoreSet puts value to tags cache.
func TagStoreSet(session *Session, ts *TagStore) (err error) {
	// set to memory cache
	tagcache.Set(ts.Puid, ts.TagProp)
	// set to database
	if affected, _ := session.InsertOne(ts); affected == 0 {
		_, err = session.ID(ts.Puid).AllCols().Omit("puid").Update(ts)
	}
	return
}

// DirStoreCat returns PUIDs list of directories where number
// of files of given category is more then given percent.
func DirStoreCat(session *Session, cat string, percent float64) (ret []Puid_t, err error) {
	const categoryCond = "(%s)/(other+video+audio+image+books+texts+packs+dir) > %f"
	err = session.Table("dir_store").Cols("puid").Where(fmt.Sprintf(categoryCond, cat, percent)).Find(&ret)
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
			var syspath, ok = pathcache.GetDir(key.(Puid_t))
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
			var syspath, ok = pathcache.GetDir(key.(Puid_t))
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

	_, err = SqlSession(func(session *Session) (res interface{}, err error) {
		if err = session.Sync(&PathStore{}, &FileStore{}, &DirStore{}, &ExifStore{}, &TagStore{}); err != nil {
			return
		}

		// fill path_store & file_store with predefined items
		var ok bool
		if ok, err = session.IsTableEmpty(&PathStore{}); err != nil {
			return
		}
		if ok {
			var ctgrpath = make([]PathStore, PUIDcache-1)
			var ctgrfile = make([]FileStore, PUIDcache-1)
			for puid, path := range CatKeyPath {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				ctgrfile[puid-1].Puid = puid
				ctgrfile[puid-1].FileProp = FileProp{
					PathProp: PathProp{
						NameVal: path,
						TypeVal: FTctgr,
					},
				}
			}
			for puid := Puid_t(len(CatKeyPath) + 1); puid < PUIDcache; puid++ {
				var path = fmt.Sprintf("<reserved%d>", puid)
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				ctgrfile[puid-1].Puid = puid
				ctgrfile[puid-1].FileProp = FileProp{
					PathProp: PathProp{
						NameVal: path,
						TypeVal: FTctgr,
					},
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
		if err = session.Where("latitude != 0").Cols("datetime", "latitude", "longitude", "altitude").Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ec := range chunk {
			var gi GpsInfo
			gi.FromProp(&ec.ExifProp)
			gpscache.Store(ec.Puid, gi)
		}
		if limit > len(chunk) {
			break
		}
	}
	return
}

// The End.
