package hms

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/schwarzlichtbezirk/wpk"
)

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
		"buildvers": BuildVers,
		"buildtime": BuildTime,
		"started":   starttime,
		"govers":    runtime.Version(),
		"os":        runtime.GOOS,
		"numcpu":    runtime.NumCPU(),
		"maxprocs":  runtime.GOMAXPROCS(0),
		"curpath":   curpath,
		"exepath":   exepath,
		"cfgpath":   ConfigPath,
		"wpkpath":   PackPath,
		"cchpath":   CachePath,
	}

	WriteOK(w, r, ret)
}

// APIHANDLER
func memusgAPI(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var ret = XmlMap{
		"buildvers":     BuildVers,
		"buildtime":     BuildTime,
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
	var session = xormStorage.NewSession()
	defer session.Close()

	var pathcount, _ = session.Count(&PathStore{})
	var dircount, _ = session.Count(&DirStore{})
	var exifcount, _ = session.Count(&ExifStore{})
	var tagcount, _ = session.Count(&TagStore{})
	var gpscount = gpscache.Len()
	var etmbcount = tmbcache.Len()

	type stat struct {
		size1 float64
		size2 float64
		num   int
	}
	var webp, jpg, png, gif stat
	thumbpkg.Enum(func(fkey string, ts *wpk.TagsetRaw) bool {
		var l = float64(ts.Size())
		if str, ok := ts.TagStr(wpk.TIDmime); ok {
			var s *stat
			switch MimeVal[str] {
			case MimeGif:
				s = &gif
			case MimePng:
				s = &png
			case MimeJpeg:
				s = &jpg
			case MimeWebp:
				s = &webp
			default:
				panic(fmt.Sprintf("unexpected MIME type in cache %s", str))
			}
			s.size1 += l
			s.size2 += l * l
			s.num++
		}
		return true
	})

	var isocount int
	for _, cc := range IsoCaches {
		isocount += len(cc.cache)
	}

	var ftpcount int
	for _, cc := range FtpCaches {
		ftpcount += len(cc.cache)
	}

	var sftpcount int
	for _, cc := range SftpCaches {
		sftpcount += len(cc.cache)
	}

	var ret = XmlMap{
		"pathcount":    pathcount,
		"dircount":     dircount,
		"exifcount":    exifcount,
		"tagcount":     tagcount,
		"gpscount":     gpscount,
		"etmbcount":    etmbcount,
		"mtmbcount":    gif.num + png.num + jpg.num + webp.num,
		"mtmbsumsize1": gif.size1 + png.size1 + jpg.size1 + webp.size1,
		"mtmbsumsize2": gif.size2 + png.size2 + jpg.size2 + webp.size2,
		"webpnum":      webp.num,
		"webpsumsize1": webp.size1,
		"webpsumsize2": webp.size2,
		"jpgnum":       jpg.num,
		"jpgsumsize1":  jpg.size1,
		"jpgsumsize2":  jpg.size2,
		"pngnum":       png.num,
		"pngsumsize1":  png.size1,
		"pngsumsize2":  png.size2,
		"gifnum":       gif.num,
		"gifsumsize1":  gif.size1,
		"gifsumsize2":  gif.size2,
		"isocount":     isocount,
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

		List []LogStore `json:"list" yaml:"list" xml:"list>item"`
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
			if from.After(h.Value.(LogStore).Time) {
				num = i
				break
			}
		}
	}
	ret.List = make([]LogStore, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		ret.List[num-i-1] = h.Value.(LogStore)
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
	if acc = prflist.ByID(aid); acc == nil {
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

	var session = xormStorage.NewSession()
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

	var ext = GetFileExt(syspath)
	if IsTypeEXIF(ext) {
		var ep ExifProp
		if err = ep.Extract(syspath); err != nil {
			WriteError(w, r, http.StatusNoContent, err, AECtagsnoexif)
			return
		}
		ret.Prop = &ep
	} else if IsTypeID3(ext) {
		var tp TagProp
		if err = tp.Extract(syspath); err != nil {
			WriteError(w, r, http.StatusNoContent, err, AECtagsnoid3)
			return
		}
		ret.Prop = &tp
	} else {
		WriteError(w, r, http.StatusNoContent, ErrNoData, AECtagsnotags)
		return
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
	if acc = prflist.ByID(aid); acc == nil {
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

	var session = xormStorage.NewSession()
	defer session.Close()

	var fpath = ToSlash(arg.Path)
	var syspath string
	var fi fs.FileInfo
	if fi, _ = StatFile(fpath); fi != nil {
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
		if fi, err = StatFile(syspath); err != nil {
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
