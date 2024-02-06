package hms

import (
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	cfg "github.com/schwarzlichtbezirk/hms/config"
	"github.com/schwarzlichtbezirk/wpk"
)

// save server start time
var starttime = time.Now()

// Check service response.
func SpiPing(c *gin.Context) {
	var ret = gin.H{
		"message": "pong",
	}
	RetOk(c, ret)
}

// APIHANDLER
func reloadAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}

	if err = OpenPackage(); err != nil {
		WriteError500(w, r, err, SEC_reload_load)
		return
	}
	if err = LoadTemplates(); err != nil {
		WriteError500(w, r, err, SEC_reload_tmpl)
		return
	}

	WriteOK(w, r, nil)
}

// Static service system information.
func SpiServInfo(c *gin.Context) {
	var ret = gin.H{
		"buildvers": cfg.BuildVers,
		"buildtime": cfg.BuildTime,
		"started":   starttime.Format(time.RFC3339),
		"govers":    runtime.Version(),
		"os":        runtime.GOOS,
		"numcpu":    runtime.NumCPU(),
		"maxprocs":  runtime.GOMAXPROCS(0),
		"exepath":   cfg.ExePath,
		"cfgpath":   cfg.CfgPath,
		"pkgpath":   cfg.PkgPath,
		"tmbpath":   cfg.TmbPath,
	}
	RetOk(c, ret)
}

// Memory usage footprint.
func SpiMemUsage(c *gin.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var ret = XmlMap{
		"buildvers":     cfg.BuildVers,
		"buildtime":     cfg.BuildTime,
		"running":       time.Since(starttime) / time.Millisecond,
		"heapalloc":     mem.HeapAlloc,
		"heapsys":       mem.HeapSys,
		"totalalloc":    mem.TotalAlloc,
		"nextgc":        mem.NextGC,
		"numgc":         mem.NumGC,
		"pausetotalns":  mem.PauseTotalNs,
		"gccpufraction": mem.GCCPUFraction,
	}
	RetOk(c, ret)
}

// Get caches state snapshot.
func SpiCachesInfo(c *gin.Context) {
	var session = XormStorage.NewSession()
	defer session.Close()

	var (
		pathcount, _ = session.Count(&PathStore{})
		dircount, _  = session.Count(&DirStore{})
		exifcount, _ = session.Count(&ExifStore{})
		tagcount, _  = session.Count(&Id3Store{})
		gpscount     = GpsCache.Len()
		etmbcount    = etmbcache.Len()
		etmbsize     = CacheSize(etmbcache)
		imgcount     = imgcache.Len()
		imgsize      = CacheSize(imgcache)
	)

	var (
		size1 float64
		size2 float64
		num   int
	)
	ThumbPkg.Enum(func(fkey string, ts wpk.TagsetRaw) bool {
		var l = float64(ts.Size())
		size1 += l
		size2 += l * l
		num++
		return true
	})

	var isocount, davcount, ftpcount, sftpcount int
	var keys = JP.Keys()
	for _, key := range keys {
		var jc = JP.GetCache(key)
		if len(key) >= 4 && ToLower(key[len(key)-4:]) == ".iso" {
			isocount += jc.Count()
		} else if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
			davcount += jc.Count()
		} else if strings.HasPrefix(key, "ftp://") {
			ftpcount += jc.Count()
		} else if strings.HasPrefix(key, "sftp://") {
			sftpcount += jc.Count()
		}
	}

	var ret = XmlMap{
		"pathcount":    pathcount,
		"dircount":     dircount,
		"exifcount":    exifcount,
		"tagcount":     tagcount,
		"gpscount":     gpscount,
		"etmbcount":    etmbcount,
		"etmbsize":     etmbsize,
		"imgcount":     imgcount,
		"imgsize":      imgsize,
		"mtmbcount":    num,
		"mtmbsumsize1": size1,
		"mtmbsumsize2": size2,
		"isocount":     isocount,
		"davcount":     davcount,
		"ftpcount":     ftpcount,
		"sftpcount":    sftpcount,
	}

	RetOk(c, ret)
}

// Returns log items.
func SpiGetLog(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Num    int   `json:"num" yaml:"num" xml:"num" form:"num"`
		Unixms int64 `json:"unixms" yaml:"unixms" xml:"unixms" form:"unixms"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []cfg.LogStore `json:"list" yaml:"list" xml:"list>item"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_getlog_nobind, err)
		return
	}

	var size = Log.Size()
	var num = arg.Num
	if num <= 0 || num > size {
		num = size
	}
	var from = time.UnixMilli(arg.Unixms)

	if !from.IsZero() {
		var h = Log.Ring()
		for i := 0; i < num; i++ {
			if from.After(h.Value.(cfg.LogStore).Time) {
				num = i
				break
			}
		}
	}
	ret.List = make([]cfg.LogStore, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		ret.List[num-i-1] = h.Value.(cfg.LogStore)
		h = h.Prev()
	}

	RetOk(c, ret)
}

// APIHANDLER
func tagsAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Prop any `json:"prop" yaml:"prop" xml:"prop"`
	}

	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		WriteError400(w, r, ErrNoAcc, SEC_tags_noacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, SEC_tags_nodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, ok = PathStorePath(session, arg.PUID); !ok {
		WriteError400(w, r, ErrNoPath, SEC_tags_badpath)
		return
	}

	if Hidden.Fits(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, SEC_tags_hidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, SEC_tags_access)
		return
	}

	var buf StoreBuf
	buf.Init(1) // flush on every push

	if ret.Prop, _, err = TagsExtract(syspath, session, &buf, &ExtStat{}, false); err != nil {
		if !errors.Is(err, io.EOF) {
			WriteError500(w, r, err, SEC_tags_extract)
			return
		}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ispathAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Valid bool `json:"valid" yaml:"valid" xml:"valid"`
		IsDir bool `json:"isdir" yaml:"isdir" xml:"isdir"`
		Space bool `json:"space" yaml:"space" xml:"space"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		WriteError400(w, r, ErrNoAcc, SEC_ispath_noacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, SEC_ispath_deny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, SEC_ispath_nodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var fpath = ToSlash(arg.Path)
	var syspath string
	var fi fs.FileInfo
	if fi, _ = JP.Stat(fpath); fi != nil {
		syspath = path.Clean(fpath)
		// append slash to disk root to prevent open current dir on this disk
		if syspath[len(syspath)-1] == ':' { // syspath here is always have non zero length
			syspath += "/"
		}
	} else {
		if syspath, _, err = UnfoldPath(session, fpath); err != nil {
			ret.Valid = false
			WriteOK(w, r, &ret)
			return
		}
		if fi, err = JP.Stat(syspath); err != nil {
			ret.Valid = false
			WriteOK(w, r, &ret)
			return
		}
	}

	if Hidden.Fits(syspath) {
		ret.Valid = false
		WriteOK(w, r, &ret)
		return
	}

	ret.Valid = true
	ret.IsDir = fi.IsDir()
	ret.Space = acc.PathAdmin(syspath)
	WriteOK(w, r, &ret)
}

// The End.
