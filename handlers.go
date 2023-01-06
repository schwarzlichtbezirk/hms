package hms

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/wpk"
)

//////////////////////////
// API request handlers //
//////////////////////////

// APIHANDLER
func pageHandler(pref, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var alias = pagealias[name]
		var content, ok = pagecache[pref+"/"+alias]
		if !ok {
			WriteError(w, r, http.StatusNotFound, ErrNotFound, AECpageabsent)
		}
		if name == "main" {
			go func() {
				var chunks = strings.Split(r.URL.Path, "/")
				var pos = 1
				if len(chunks) > pos && chunks[pos] == "dev" {
					pos++
				}
				var aid = cfg.DefAccID
				if len(chunks) > pos && len(chunks[pos]) > 2 && chunks[pos][:2] == "id" {
					if u64, err := strconv.ParseUint(chunks[pos][2:], 10, 32); err == nil {
						aid = ID_t(u64)
					}
				}
				usermsg <- UsrMsg{r, "page", aid}
			}()
		}

		WriteHTMLHeader(w)
		http.ServeContent(w, r, alias, starttime, bytes.NewReader(content))
	}
}

// Hands out converted media files if them can be cached.
func fileHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// get arguments
	var vars = mux.Vars(r)
	if vars == nil {
		panic("bad route for URL " + r.URL.Path)
	}
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECmedianoaid)
		return
	}
	var media bool
	if s := r.FormValue("media"); len(s) > 0 {
		if media, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, AECmediabadmedia)
			return
		}
	}
	var hd bool
	if s := r.FormValue("hd"); len(s) > 0 {
		if hd, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, AECmediabadhd)
			return
		}
	}

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}
	var fpath = strings.Join(chunks[3:], "/")

	var session = xormStorage.NewSession()
	defer session.Close()

	var syspath string
	var puid Puid_t
	if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
		WriteError400(w, r, err, AECmediabadpath)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECmedianoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	if strings.HasPrefix(syspath, "http://") || strings.HasPrefix(syspath, "https://") {
		http.Redirect(w, r, syspath, http.StatusMovedPermanently)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECmediahidden)
		return
	}

	if !prf.PathAccess(syspath, auth == prf) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmediaaccess)
		return
	}

	var grp = GetFileGroup(syspath)
	if hd && grp == FGimage {
		var md MediaData
		if md, err = HdCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, AECmediahdgone)
				return
			}
			if !errors.Is(err, ErrNotHD) {
				WriteError500(w, r, err, AECmediahdfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, AECmediahdnocnt)
				return
			}

			if uid, ok := r.Context().Value(ctxkey("UID")).(uint64); ok {
				openlog <- OpenStore{
					UID:     uid,
					AID:     aid,
					Path:    syspath,
					Time:    time.Now(),
					Latency: -1,
				}
			}
			go func() {
				if _, ok := r.Header["If-Range"]; !ok {
					Log.Infof("id%d: media-hd %s", prf.ID, path.Base(syspath))
					// not partial content
					usermsg <- UsrMsg{r, "file", puid}
				} else {
					// update statistics for partial content
					userajax <- r
				}
			}()
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, syspath, starttime, bytes.NewReader(md.Data))
			return
		}
	}

	if media && grp == FGimage {
		var md MediaData
		if md, err = MediaCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, AECmediamedgone)
				return
			}
			if !errors.Is(err, ErrUncacheable) {
				WriteError(w, r, http.StatusNotFound, err, AECmediamedfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, AECmediamednocnt)
				return
			}

			if uid, ok := r.Context().Value(ctxkey("UID")).(uint64); ok {
				openlog <- OpenStore{
					UID:     uid,
					AID:     aid,
					Path:    syspath,
					Time:    time.Now(),
					Latency: -1,
				}
			}
			go func() {
				if _, ok := r.Header["If-Range"]; !ok {
					Log.Infof("id%d: media %s", prf.ID, path.Base(syspath))
					// not partial content
					usermsg <- UsrMsg{r, "file", puid}
				} else {
					// update statistics for partial content
					userajax <- r
				}
			}()
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, syspath, starttime, bytes.NewReader(md.Data))
			return
		}
	}

	if uid, ok := r.Context().Value(ctxkey("UID")).(uint64); ok {
		openlog <- OpenStore{
			UID:     uid,
			AID:     aid,
			Path:    syspath,
			Time:    time.Now(),
			Latency: -1,
		}
	}
	go func() {
		if _, ok := r.Header["If-Range"]; !ok {
			Log.Infof("id%d: serve %s", prf.ID, path.Base(syspath))
			// not partial content
			usermsg <- UsrMsg{r, "file", puid}
		} else {
			// update statistics for partial content
			userajax <- r
		}
	}()

	var content io.ReadSeekCloser
	if content, err = OpenFile(syspath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			WriteError(w, r, http.StatusGone, err, AECmediafilegone)
			return
		}
		WriteError500(w, r, err, AECmediafileopen)
		return
	}
	defer content.Close()

	WriteStdHeader(w)
	http.ServeContent(w, r, syspath, starttime, content)
}

// Hands out embedded thumbnails for given files if any.
func etmbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// get arguments
	var vars = mux.Vars(r)
	if vars == nil {
		panic("bad route for URL " + r.URL.Path)
	}
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECetmbnoaid)
		return
	}
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECetmbnopuid)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECetmbnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECetmbnopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECetmbhidden)
		return
	}

	if !prf.PathAccess(syspath, auth == prf) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECetmbaccess)
		return
	}

	var md MediaData
	if md, err = ExtractThmub(session, syspath); err != nil {
		if errors.Is(err, ErrNoThumb) {
			WriteError(w, r, http.StatusNoContent, err, AECetmbnotmb)
			return
		} else {
			WriteError500(w, r, err, AECetmbbadcnt)
			return
		}
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, syspath, starttime, bytes.NewReader(md.Data))
}

// Hands out cached thumbnails for given files.
func mtmbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// get arguments
	var vars = mux.Vars(r)
	if vars == nil {
		panic("bad route for URL " + r.URL.Path)
	}
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECmtmbnoaid)
		return
	}
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECmtmbnopuid)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECmtmbnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECmtmbnopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECmtmbhidden)
		return
	}

	if !prf.PathAccess(syspath, auth == prf) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmtmbaccess)
		return
	}

	var md MediaData
	if md, err = thumbpkg.GetImage(syspath); err != nil {
		WriteError500(w, r, err, AECmtmbbadcnt)
		return
	}
	if md.Mime == MimeNil {
		WriteError(w, r, http.StatusNoContent, ErrNotFound, AECmtmbnocnt)
		return
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, syspath, starttime, bytes.NewReader(md.Data))
}

// Hands out thumbnails for given files if them cached.
func tileHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// get arguments
	var vars = mux.Vars(r)
	if vars == nil {
		panic("bad route for URL " + r.URL.Path)
	}
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECtilenoaid)
		return
	}
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECtilenopuid)
		return
	}
	var wdh, _ = strconv.Atoi(vars["wdh"])
	var hgt, _ = strconv.Atoi(vars["hgt"])
	if wdh == 0 || hgt == 0 {
		WriteError400(w, r, ErrArgNoRes, AECtilebadres)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECtilenoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECtilenopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECtilehidden)
		return
	}

	if !prf.PathAccess(syspath, auth == prf) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECtileaccess)
		return
	}

	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)
	var md MediaData
	if md, err = tilespkg.GetImage(tilepath); err != nil {
		WriteError500(w, r, err, AECtilebadcnt)
		return
	}
	if md.Mime == MimeNil {
		WriteError(w, r, http.StatusNoContent, ErrNotFound, AECtilenocnt)
		return
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, syspath, starttime, bytes.NewReader(md.Data))
}

// APIHANDLER
func pingAPI(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	WriteStdHeader(w)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// APIHANDLER
func reloadAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	_ = auth
	var err error

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
		"builddate": BuildDate,
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
		"builddate":     BuildDate,
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
	var gpscount = gpscache.Count()
	var etmbcount = tmbcache.Len()

	type stat struct {
		size1 float64
		size2 float64
		num   int
	}
	var jpg, png, gif stat
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
			default:
				panic(fmt.Sprintf("unexpected MIME type in cache %s", str))
			}
			s.size1 += l
			s.size2 += l * l
			s.num++
		}
		return true
	})

	var ret = XmlMap{
		"pathcount":    pathcount,
		"dircount":     dircount,
		"exifcount":    exifcount,
		"tagcount":     tagcount,
		"gpscount":     gpscount,
		"etmbcount":    etmbcount,
		"mtmbcount":    gif.num + png.num + jpg.num,
		"mtmbsumsize1": gif.size1 + png.size1 + jpg.size1,
		"mtmbsumsize2": gif.size2 + png.size2 + jpg.size2,
		"jpgnum":       jpg.num,
		"jpgsumsize1":  jpg.size1,
		"jpgsumsize2":  jpg.size2,
		"pngnum":       png.num,
		"pngsumsize1":  png.size1,
		"pngsumsize2":  png.size2,
		"gifnum":       gif.num,
		"gifsumsize1":  gif.size1,
		"gifsumsize2":  gif.size2,
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

	ret.List = make([]LogStore, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		ret.List[num-i-1] = h.Value.(LogStore)
		h = h.Prev()
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func propAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Path string  `json:"path" yaml:"path" xml:"path"`
		Name string  `json:"shrname" yaml:"shrname" xml:"shrname"`
		Prop FileKit `json:"prop" yaml:"prop" xml:"prop"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECpropnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECpropnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECpropnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath string
	var ok bool
	if syspath, ok = PathStorePath(session, arg.PUID); !ok {
		WriteError400(w, r, ErrNoPath, AECpropbadpath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECprophidden)
		return
	}

	var shrpath, base, cg = prf.GetSharePath(session, syspath, auth == prf)
	if cg.IsZero() && syspath != CPshares {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECpropaccess)
		return
	}
	ret.Path = shrpath
	ret.Name = path.Base(base)

	var fi fs.FileInfo
	if fi, err = StatFile(syspath); err != nil {
		WriteError500(w, r, err, AECpropbadstat)
		return
	}
	ret.Prop.Setup(session, syspath, fi)
	ret.Prop.Shared = prf.IsShared(syspath)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ispathAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECispathnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECispathnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECispathnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECispathdeny)
		return
	}

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
			WriteError400(w, r, err, AECispathbadpath)
			return
		}
		if fi, err = StatFile(syspath); err != nil {
			WriteError(w, r, http.StatusNotFound, http.ErrMissingFile, AECispathmiss)
			return
		}
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECispathhidden)
		return
	}

	var fk FileKit
	fk.Setup(session, syspath, fi)
	fk.Shared = prf.IsShared(syspath)
	WriteOK(w, r, &fk)
}

// APIHANDLER
func shraddAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Shared bool `json:"shared" yaml:"shared" xml:"shared"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECshraddnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshraddnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECshraddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECshradddeny)
		return
	}

	var syspath, ok = PathStorePath(session, arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECshraddnopath)
	}
	if !prf.PathAdmin(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECshraddaccess)
		return
	}

	ret.Shared = prf.AddShare(session, syspath)
	Log.Infof("id%d: add share '%s' as %s", prf.ID, syspath, arg.PUID)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func shrdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECshrdelnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshrdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECshrdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECshrdeldeny)
		return
	}

	if ret.Deleted = prf.DelShare(arg.PUID); ret.Deleted {
		Log.Infof("id%d: delete share %s", prf.ID, arg.PUID)
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func drvaddAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECdrvaddnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvaddnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvaddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvadddeny)
		return
	}

	var fpath = ToSlash(arg.Path)
	var syspath string
	var puid Puid_t
	var fi fs.FileInfo
	if fi, _ = StatFile(fpath); fi != nil {
		syspath = path.Clean(fpath)
		// append slash to disk root to prevent open current dir on this disk
		if syspath[len(syspath)-1] == ':' { // syspath here is always have non zero length
			syspath += "/"
		}
		puid = PathStoreCache(session, syspath)
	} else {
		if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
			WriteError400(w, r, err, AECdrvaddbadpath)
			return
		}
		if fi, err = StatFile(syspath); err != nil {
			WriteError(w, r, http.StatusNotFound, http.ErrMissingFile, AECdrvaddmiss)
			return
		}
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECdrvaddhidden)
		return
	}

	if prf.RootIndex(syspath) >= 0 {
		WriteOK(w, r, nil)
		return
	}

	var fk FileKit
	fk.PUID = puid
	fk.Shared = prf.IsShared(syspath)
	fk.Name = path.Base(syspath)
	fk.Type = FTdrv
	fk.Size = fi.Size()
	fk.Time = fi.ModTime()

	prf.mux.Lock()
	prf.Roots = append(prf.Roots, syspath)
	prf.mux.Unlock()

	WriteOK(w, r, &fk)
}

// APIHANDLER
func drvdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid uint64
	if aid, err = strconv.ParseUint(vars["aid"], 10, 64); err != nil {
		WriteError400(w, r, err, AECdrvdelnoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvdelnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvdeldeny)
		return
	}

	var syspath, ok = PathStorePath(session, arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECdrvdelnopath)
	}

	var i int
	if i = prf.RootIndex(syspath); i >= 0 {
		prf.mux.Lock()
		prf.Roots = append(prf.Roots[:i], prf.Roots[i+1:]...)
		prf.mux.Unlock()
	}

	ret.Deleted = i >= 0
	WriteOK(w, r, &ret)
}

// The End.
