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
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/fsys"

	"github.com/chai2010/webp"
	"github.com/disintegration/gift"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

const (
	dirfile = "storage.sqlite"
	userlog = "userlog.sqlite"

	tmbfile = "thumb.wpt"
	tilfile = "tiles.wpt"
)

var (
	S2B      = wpk.S2B
	B2S      = wpk.B2S
	JoinPath = wpk.JoinPath
	ToSlash  = wpk.ToSlash
	ToLower  = wpk.ToLower
	Cfg      = cfg.Cfg
	Log      = cfg.Log
)

type (
	Session = xorm.Session
)

// package caches
var (
	// cache with images thumbnails which are placed in box 256x256.
	ThumbPkg *FileCache
	// cache with images tiles, size of each tile is placed as sufix
	// of path in format "full/path/to/file.ext?144x108".
	TilesPkg *FileCache
)

var XormStorage *xorm.Engine

// Error messages
var (
	ErrNoPUID      = errors.New("file with given puid not found")
	ErrUncacheable = errors.New("file format is uncacheable")
	ErrNotHD       = errors.New("image dimensions does not fit to full HD")
	ErrNotDisk     = errors.New("file is not image of supported format")
	ErrNoMTime     = errors.New("modify time tag does not found")
	ErrNoMime      = errors.New("MIME tag does not found")
	ErrEmptyExif   = errors.New("EXIF metadata is empty")
	ErrEmptyID3    = errors.New("ID3 metadata is empty")
)

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

// SqlSession execute sql wrapped in a single session.
func SqlSession(f func(*Session) (any, error)) (any, error) {
	var session = XormStorage.NewSession()
	defer session.Close()
	return f(session)
}

// GpsInfo describes GPS-data from the photos:
// latitude, longitude, altitude and creation time.
type GpsInfo struct {
	DateTime  time.Time `xorm:"DateTime" json:"time" yaml:"time" xml:"time,attr"` // photo creation date/time in Unix milliseconds
	Latitude  float64   `json:"lat" yaml:"lat" xml:"lat,attr"`
	Longitude float64   `json:"lon" yaml:"lon" xml:"lon,attr"`
	Altitude  float32   `json:"alt,omitempty" yaml:"alt,omitempty" xml:"alt,omitempty,attr"`
}

// FromProp fills fields with values from ExifProp.
func (gi *GpsInfo) FromProp(tp *ExifProp) {
	gi.DateTime = tp.DateTime
	gi.Latitude = tp.Latitude
	gi.Longitude = tp.Longitude
	gi.Altitude = tp.Altitude
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
	ExtStore  Store[ExtProp]
	ExifStore Store[ExifProp]
	Id3Store  Store[Id3Prop]
)

var (
	PathCache = NewBimap[Puid_t, string]()   // Bidirectional map for PUIDs and system paths.
	gpscache  = NewCache[Puid_t, GpsInfo]()  // FIFO cache with GPS coordinates.
	tilecache = NewCache[Puid_t, TileProp]() // FIFO cache with set of available tiles.
	extcache  = NewCache[Puid_t, ExtProp]()  // FIFO cache with set of base extension properties.

	etmbcache = NewCache[Puid_t, MediaData]() // FIFO cache with files embedded thumbnails.
	imgcache  = NewCache[Puid_t, MediaData]() // FIFO cache with converted to HD resolution images, processed media files and embedded thumbnails.

	pubkcache = NewCache[[32]byte, TempCell[struct{}]]() // LRU cache with public keys.
)

const (
	PuidTmb Puid_t = 0x2000_0000_0000_0000 // Embedded thumbnails qualifier
	PuidHD  Puid_t = 0x4000_0000_0000_0000 // HD-dimensions qualifier
	PuidImg Puid_t = 0x8000_0000_0000_0000 // Converted to acceptable browser format qualifier
)

// PathStorePUID returns cached PUID for specified system path.
func PathStorePUID(session *Session, fpath string) (puid Puid_t, ok bool) {
	// try to get from memory cache
	if puid, ok = PathCache.GetRev(fpath); ok {
		return
	}
	// try to get from database
	var val uint64
	if ok, _ = session.Table("path_store").Cols("puid").Where("path=?", fpath).Get(&val); ok { // skip errors
		puid = Puid_t(val)
		PathCache.Set(puid, fpath) // update cache
		return
	}
	return
}

// PathStorePath returns cached system path of specified PUID (path unique identifier).
func PathStorePath(session *Session, puid Puid_t) (fpath string, ok bool) {
	// try to get from memory cache
	if fpath, ok = PathCache.GetDir(puid); ok {
		return
	}
	// try to get from database
	if ok, _ = session.Table("path_store").Cols("path").Where("puid=?", puid).Get(&fpath); ok { // skip errors
		PathCache.Set(puid, fpath) // update cache
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
	PathCache.Set(puid, fpath)
	return
}

// DirStoreSet puts value to directories cache.
func DirStoreSet(session *Session, puid Puid_t, dp DirProp) (err error) {
	var dst = &DirStore{
		Puid: puid,
		Prop: dp,
	}
	// set to database
	if affected, _ := session.InsertOne(dst); affected == 0 {
		_, err = session.ID(dst.Puid).AllCols().Omit("puid").Update(dst)
	}
	return
}

// ExtStoreGet returns value from embedded info database.
func ExtStoreGet(session *Session, puid Puid_t) (xp ExtProp, ok bool) {
	// try to get from database
	var xst ExtStore
	if ok, _ = session.ID(puid).Get(&xst); ok {
		xp = xst.Prop
		return
	}
	return
}

// ExtStoreSet puts value to embedded info database.
func ExtStoreSet(session *Session, puid Puid_t, xp ExtProp) (err error) {
	var xst = &ExtStore{
		Puid: puid,
		Prop: xp,
	}
	// set to database
	if affected, _ := session.InsertOne(xst); affected == 0 {
		_, err = session.ID(xst.Puid).AllCols().Omit("puid").Update(xst)
	}
	return
}

func GpsCachePut(puid Puid_t, tp ExifProp) {
	if tp.Latitude != 0 || tp.Longitude != 0 {
		var gi GpsInfo
		gi.FromProp(&tp)
		gpscache.Poke(puid, gi)
	}
}

// ExifStoreGet returns value from EXIF database.
func ExifStoreGet(session *Session, puid Puid_t) (tp ExifProp, ok bool) {
	// try to get from database
	var est ExifStore
	if ok, _ = session.ID(puid).Get(&est); ok {
		tp = est.Prop
		return
	}
	return
}

// ExifStoreSet puts value to EXIF database.
func ExifStoreSet(session *Session, puid Puid_t, tp ExifProp) (err error) {
	var est = &ExifStore{
		Puid: puid,
		Prop: tp,
	}
	// set to GPS cache
	GpsCachePut(est.Puid, est.Prop)
	// set to database
	if affected, _ := session.InsertOne(est); affected == 0 {
		_, err = session.ID(est.Puid).AllCols().Omit("puid").Update(est)
	}
	return
}

// Id3StoreGet returns value from tags database.
func Id3StoreGet(session *Session, puid Puid_t) (tp Id3Prop, ok bool) {
	// try to get from database
	var tst Id3Store
	if ok, _ = session.ID(puid).Get(&tst); ok {
		tp = tst.Prop
		return
	}
	return
}

// Id3StoreSet puts value to tags database.
func Id3StoreSet(session *Session, puid Puid_t, tp Id3Prop) (err error) {
	var tst = &Id3Store{
		Puid: puid,
		Prop: tp,
	}
	// set to database
	if affected, _ := session.InsertOne(tst); affected == 0 {
		_, err = session.ID(tst.Puid).AllCols().Omit("puid").Update(tst)
	}
	return
}

var tmbmux, imgmux sync.Mutex

func ThumbCacheTrim() {
	tmbmux.Lock()
	defer tmbmux.Unlock()
	var size = CacheSize(etmbcache)
	var sum int64
	etmbcache.Until(func(puid Puid_t, md MediaData) bool {
		sum += md.Size()
		return float32(size-sum)/1048576 > Cfg.ThumbCacheMaxSize
	})
}

func ImgCacheTrim() {
	imgmux.Lock()
	defer imgmux.Unlock()
	var size = CacheSize(imgcache)
	var sum int64
	imgcache.Until(func(puid Puid_t, md MediaData) bool {
		sum += md.Size()
		return float32(size-sum)/1048576 > Cfg.ImgCacheMaxSize
	})
}

// MediaCacheGet returns media file with given PUID converted to acceptable
// for browser format from memory cache.
func MediaCacheGet(session *Session, puid Puid_t) (md MediaData, err error) {
	var ok bool
	if md, ok = imgcache.Peek(puid | PuidImg); ok {
		return
	}

	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
		err = ErrNoPUID
		return // file path not found
	}

	var ext = GetFileExt(syspath)
	if IsTypeNativeImg(ext) || !IsTypeDecoded(ext) {
		err = ErrUncacheable
		return // uncacheable type
	}

	var file fs.File
	if file, err = JP.Open(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var src image.Image
	if src, _, err = image.Decode(file); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}

	if md.Data, err = webp.EncodeRGBA(src, Cfg.MediaWebpQuality); err != nil {
		return // can not write webp
	}
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}
	imgcache.Poke(puid|PuidImg, md)
	go ImgCacheTrim()
	return
}

// HdCacheGet returns image with given PUID converted to HD resolution
// from memory cache.
func HdCacheGet(session *Session, puid Puid_t) (md MediaData, err error) {
	var ok bool
	if md, ok = imgcache.Peek(puid | PuidHD); ok {
		return
	}

	var syspath string
	if syspath, ok = PathStorePath(session, puid); !ok {
		err = ErrNoPUID
		return // file path not found
	}

	var file RFile
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var imc image.Config
	if imc, _, err = image.DecodeConfig(file); err != nil {
		return // can not recognize format or decode config
	}
	if float32(imc.Width*imc.Height+5e5)/1e6 > Cfg.ImageMaxMpx {
		err = ErrTooBig
		return // file is too big
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return // can not seek to start
	}

	var wdh, hgt int
	if imc.Width > imc.Height {
		wdh, hgt = Cfg.HDResolution[0], Cfg.HDResolution[1]
	} else {
		wdh, hgt = Cfg.HDResolution[1], Cfg.HDResolution[0]
	}
	if imc.Width <= wdh && imc.Height <= hgt {
		err = ErrNotHD
		return // does not fit to HD
	}

	// try to extract orientation from EXIF
	var orientation = OrientNormal
	if tp, ok := ExifStoreGet(session, puid); ok && tp.Orientation > 0 {
		orientation = tp.Orientation
	} else if tp, err = ExifExtract(session, file, puid); err == nil && tp.Orientation > 0 {
		orientation = tp.Orientation
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return
	}

	var src, dst image.Image
	if src, _, err = image.Decode(file); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
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

	if md.Data, err = webp.EncodeRGBA(dst, Cfg.HDWebpQuality); err != nil {
		return // can not write webp
	}
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}
	imgcache.Poke(puid|PuidHD, md)
	go ImgCacheTrim()
	return
}

// FileCache describes package with cache functionality.
// Package splitted in two files - tags table file and
// data file with cached nested files.
type FileCache struct {
	*wpk.Package
	wpt wpk.WriteSeekCloser // package tags part
	wpf wpk.WriteSeekCloser // package files part
}

// InitCacheWriter opens existing cache with given file path placed in
// cache directory, or creates new cache file if no one found.
func InitCacheWriter(fpath string) (fc *FileCache, d time.Duration, err error) {
	var t0 = time.Now()
	defer func() {
		d = time.Since(t0)
	}()

	var pkgpath = wpk.MakeTagsPath(fpath)
	var datpath = wpk.MakeDataPath(fpath)
	fc = &FileCache{
		Package: wpk.NewPackage(),
	}
	defer func() {
		if err != nil {
			if fc.wpt != nil {
				fc.wpt.Close()
				fc.wpt = nil
			}
			if fc.wpf != nil {
				fc.wpf.Close()
				fc.wpf = nil
			}
		}
	}()

	var ok, _ = cfg.FileExists(pkgpath)
	if fc.wpt, err = os.OpenFile(pkgpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if fc.wpf, err = os.OpenFile(datpath, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		return
	}
	if ok {
		var r io.ReadSeekCloser
		if r, err = os.Open(pkgpath); err != nil {
			return
		}
		defer r.Close()

		if err = fc.OpenStream(r); err != nil {
			return
		}

		if err = fc.Append(fc.wpt, fc.wpf); err != nil {
			return
		}
		fc.Package.SetInfo(wpk.CopyTagset(fc.Package.GetInfo()).
			Set(wpk.TIDmtime, wpk.UnixmsTag(t0)))
	} else {
		fc.Init(&wpk.Header{})

		if err = fc.Begin(fc.wpt, fc.wpf); err != nil {
			return
		}
		fc.Package.SetInfo(wpk.TagsetRaw{}.
			Put(wpk.TIDlabel, wpk.StrTag(path.Base(fpath))).
			Put(wpk.TIDmtime, wpk.UnixmsTag(t0)).
			Put(wpk.TIDbtime, wpk.UnixmsTag(t0)))
	}
	if fc.Tagger, err = fsys.MakeTagger(datpath); err != nil {
		return
	}
	return
}

// Sync writes actual file tags table and true signature with settings.
func (fc *FileCache) Sync() error {
	return fc.Package.Sync(fc.wpt, fc.wpf)
}

// Close saves actual tags table and closes opened cache.
func (fc *FileCache) Close() (err error) {
	if et := fc.Sync(); et != nil && err == nil {
		err = et
	}
	if et := fc.wpt.Close(); et != nil && err == nil {
		err = et
	}
	if et := fc.wpf.Close(); et != nil && err == nil {
		err = et
	}
	fc.wpt, fc.wpf = nil, nil
	return
}

// GetFile extracts file from the cache with given file name.
func (fc *FileCache) GetFile(fpath string) (file wpk.RFile, mime string, t time.Time, err error) {
	if ts, ok := fc.GetTagset(fpath); ok {
		if t, ok = ts.TagTime(wpk.TIDmtime); !ok {
			err = ErrNoMTime
			return
		}
		if mime, ok = ts.TagStr(wpk.TIDmime); !ok {
			err = ErrNoMime
			return
		}
		if file, err = fc.OpenTagset(ts); err != nil {
			return
		}
	}
	return
}

// GetData extracts file from the cache with given file name.
func (fc *FileCache) GetData(fpath string) (md MediaData, err error) {
	if ts, ok := fc.GetTagset(fpath); ok {
		var t time.Time
		if t, ok = ts.TagTime(wpk.TIDmtime); !ok {
			err = ErrNoMTime
			return
		}
		md.Time = t

		var mime string
		if mime, ok = ts.TagStr(wpk.TIDmime); !ok {
			err = ErrNoMime
			return
		}
		if md.Mime, ok = MimeVal[mime]; !ok {
			err = ErrNotImg
			return
		}

		var file fs.File
		if file, err = fc.OpenTagset(ts); err != nil {
			return
		}
		defer file.Close()

		var data = make([]byte, ts.Size())
		if _, err = file.Read(data); err != nil {
			return
		}
		md.Data = data
	}
	return
}

// PutFile puts file to package.
func (fc *FileCache) PutFile(fpath string, md MediaData) (err error) {
	var ts wpk.TagsetRaw
	if ts, err = fc.PackData(fc.wpf, bytes.NewReader(md.Data), fpath); err != nil {
		return
	}
	if md.Time.IsZero() {
		md.Time = time.Now()
	}
	fc.SetTagset(fpath, ts.
		Put(wpk.TIDmtime, wpk.UnixmsTag(md.Time)).
		Put(wpk.TIDatime, wpk.UnixmsTag(md.Time)).
		Put(wpk.TIDmime, wpk.StrTag(MimeStr[md.Mime])))
	return
}

// PackInfo writes info to log about opened cache.
func PackInfo(fname string, pkg *wpk.Package, d time.Duration) {
	Log.Infof("package '%s': cached %d files on %d bytes, %v", fname, pkg.TagsetNum(), pkg.DataSize(), d)
}

// InitPackages opens all existing caches.
func InitPackages() (err error) {
	var d time.Duration
	if ThumbPkg, d, err = InitCacheWriter(JoinPath(cfg.TmbPath, tmbfile)); err != nil {
		err = fmt.Errorf("inits thumbnails database: %w", err)
		return
	}
	PackInfo(tmbfile, ThumbPkg.Package, d)

	if TilesPkg, d, err = InitCacheWriter(JoinPath(cfg.TmbPath, tilfile)); err != nil {
		err = fmt.Errorf("inits tiles database: %w", err)
		return
	}
	PackInfo(tilfile, TilesPkg.Package, d)

	return nil
}

// ClosePackages closes all existing caches.
func ClosePackages() (err error) {
	var err1 error
	if err1 = ThumbPkg.Close(); err1 != nil {
		err = err1
	}
	if err1 = TilesPkg.Close(); err1 != nil {
		err = err1
	}
	return
}

// InitStorage inits database caches engine.
func InitStorage() (err error) {
	if XormStorage, err = xorm.NewEngine(Cfg.XormDriverName, JoinPath(cfg.TmbPath, dirfile)); err != nil {
		return
	}
	XormStorage.SetMapper(names.GonicMapper{})
	var xlb = cfg.XormLoggerBridge{
		Logger: Log,
	}
	xlb.ShowSQL(cfg.DevMode)
	XormStorage.SetLogger(&xlb)

	_, err = SqlSession(func(session *Session) (res any, err error) {
		if err = session.Sync(&PathStore{}, &DirStore{}, &ExtStore{}, &ExifStore{}, &Id3Store{}); err != nil {
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
				PathCache.Set(puid, path)
			}
			for puid := Puid_t(len(CatKeyPath) + 1); puid < PUIDcache; puid++ {
				var path = fmt.Sprintf("<reserved%d>", puid)
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				PathCache.Set(puid, path)
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
	var session = XormStorage.NewSession()
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
			PathCache.Set(ps.Puid, ps.Path)
		}
		if limit > len(chunk) {
			break
		}
	}

	Log.Infof("loaded %d items into path cache", PathCache.Len())
	return
}

// LoadGpsCache loads all items with GPS information from EXIF table of storage into cache.
func LoadGpsCache() (err error) {
	var session = XormStorage.NewSession()
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
			gpscache.Poke(ec.Puid, gi)
		}
		if limit > len(chunk) {
			break
		}
	}

	Log.Infof("loaded %d items into GPS cache", gpscache.Len())
	return
}

// The End.
