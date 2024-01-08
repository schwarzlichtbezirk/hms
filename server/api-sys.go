package hms

import (
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	"github.com/schwarzlichtbezirk/wpk"
)

// save server start time
var starttime = time.Now()

//////////////////////////
// API request handlers //
//////////////////////////

// APIHANDLER
func pingAPI(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	WriteStdHeader(w)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// APIHANDLER
func reloadAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var uid ID_t
	if uid, err = GetAuth(r); err != nil {
		WriteRet(w, r, http.StatusUnauthorized, err)
		return
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}

	if err = OpenPackage(); err != nil {
		WriteError500(w, r, err, AECreloadload)
		return
	}
	if err = LoadTemplates(); err != nil {
		WriteError500(w, r, err, AECreloadtmpl)
		return
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func srvinfAPI(w http.ResponseWriter, r *http.Request) {
	var ret = XmlMap{
		"buildvers": cfg.BuildVers,
		"buildtime": cfg.BuildTime,
		"started":   starttime,
		"govers":    runtime.Version(),
		"os":        runtime.GOOS,
		"numcpu":    runtime.NumCPU(),
		"maxprocs":  runtime.GOMAXPROCS(0),
		"exepath":   cfg.ExePath,
		"cfgpath":   cfg.CfgPath,
		"pkgpath":   cfg.PkgPath,
		"tmbpath":   cfg.TmbPath,
	}

	WriteOK(w, r, ret)
}

// APIHANDLER
func memusgAPI(w http.ResponseWriter, r *http.Request) {
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

	WriteOK(w, r, ret)
}

// APIHANDLER
func cchinfAPI(w http.ResponseWriter, r *http.Request) {
	var session = XormStorage.NewSession()
	defer session.Close()

	var (
		pathcount, _ = session.Count(&PathStore{})
		dircount, _  = session.Count(&DirStore{})
		exifcount, _ = session.Count(&ExifStore{})
		tagcount, _  = session.Count(&Id3Store{})
		gpscount     = gpscache.Len()
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

	WriteOK(w, r, ret)
}

// APIHANDLER
func getlogAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []cfg.LogStore `json:"list" yaml:"list" xml:"list>item"`
	}

	var size = Log.Size()

	// get arguments
	var num int
	if s := r.FormValue("num"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, r, ErrArgNoNum, AECgetlogbadnum)
			return
		}
		num = int(i64)
	}
	if num <= 0 || num > size {
		num = size
	}
	var from time.Time
	if s := r.FormValue("unix"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, r, ErrArgNoTime, AECgetlogbadunix)
			return
		}
		from = time.Unix(i64, 0)
	}
	if s := r.FormValue("unixms"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, r, ErrArgNoTime, AECgetlogbadums)
			return
		}
		from = time.UnixMilli(i64)
	}

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

	WriteOK(w, r, &ret)
}

// APIHANDLER
func tagsAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Prop any `json:"prop" yaml:"prop" xml:"prop"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtagsnoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECtagsnodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	var ok bool
	if syspath, ok = PathStorePath(session, arg.PUID); !ok {
		WriteError400(w, r, ErrNoPath, AECtagsbadpath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECtagshidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECtagsaccess)
		return
	}

	var buf StoreBuf
	buf.Init(1) // flush on every push

	if ret.Prop, _, err = TagsExtract(syspath, session, &buf, &ExtStat{}, false); err != nil {
		if !errors.Is(err, io.EOF) {
			WriteError500(w, r, err, AECtagsextract)
			return
		}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ispathAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
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
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECispathnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECispathdeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECispathnodata)
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

	if acc.IsHidden(syspath) {
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