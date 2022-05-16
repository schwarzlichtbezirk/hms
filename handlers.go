package hms

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
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

	var aid uint64
	if aid, err = strconv.ParseUint(chunks[1][2:], 10, 64); err != nil {
		WriteError400(w, r, err, AECmediabadaccid)
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

	var syspath = path.Clean(strings.Join(chunks[3:], "/"))
	if syspath[0] == '.' {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmediaroot)
		return
	}
	syspath = UnfoldPath(syspath)

	var puid, ok = syspathcache.PUID(syspath)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECmedianopath)
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

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECmedianoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, r, http.StatusUnsupportedMediaType, ErrNotFile, AECmedianofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmediaaccess)
		return
	}

	var val interface{}

	if hd && grp == FGimage {
		if val, err = hdcache.Get(puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, AECmediahdgone)
				return
			}
			if !errors.Is(err, ErrNotHD) {
				WriteError500(w, r, err, AECmediahdfail)
				return
			}
		} else {
			var md *MediaData
			if md, ok = val.(*MediaData); !ok || md == nil {
				WriteError500(w, r, ErrBadMedia, AECmediahdnocnt)
				return
			}

			go func() {
				if _, ok := r.Header["If-Range"]; !ok {
					Log.Infof("id%d: media %s", prf.ID, PathBase(syspath))
					// not partial content
					usermsg <- UsrMsg{r, "file", puid}
				} else {
					// update statistics for partial content
					userajax <- r
				}
			}()
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, puid.String(), starttime, bytes.NewReader(md.Data))
			return
		}
	}

	if media && grp == FGimage {
		if val, err = mediacache.Get(puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, AECmediamedgone)
				return
			}
			if !errors.Is(err, ErrUncacheable) {
				WriteError(w, r, http.StatusNotFound, err, AECmediamedfail)
				return
			}
		} else {
			var md *MediaData
			if md, ok = val.(*MediaData); !ok || md == nil {
				WriteError500(w, r, ErrBadMedia, AECmediamednocnt)
				return
			}

			go func() {
				if _, ok := r.Header["If-Range"]; !ok {
					Log.Infof("id%d: media %s", prf.ID, PathBase(syspath))
					// not partial content
					usermsg <- UsrMsg{r, "file", puid}
				} else {
					// update statistics for partial content
					userajax <- r
				}
			}()
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, puid.String(), starttime, bytes.NewReader(md.Data))
			return
		}
	}

	go func() {
		if _, ok := r.Header["If-Range"]; !ok {
			Log.Infof("id%d: serve %s", prf.ID, PathBase(syspath))
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
	http.ServeContent(w, r, syspath, TimeJS(fp.Time()), content)
}

// Hands out thumbnails for given files if them cached.
func thumbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var chunks = mux.Vars(r)
	if chunks == nil {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid, _ = strconv.ParseUint(chunks["aid"], 10, 64)

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECthumbnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var puid Puid_t
	if err = puid.Set(chunks["puid"]); err != nil {
		WriteError400(w, r, err, AECthumbnopuid)
		return
	}
	var syspath, ok = syspathcache.Path(puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECthumbnopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECthumbhidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECthumbnoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, r, http.StatusUnsupportedMediaType, ErrNotFile, AECthumbnofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECthumbaccess)
		return
	}

	var md *MediaData
	if md, err = FindTmb(fp, syspath); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECthumbabsent)
		return
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, puid.String(), starttime, bytes.NewReader(md.Data))
}

// Hands out thumbnails for given files if them cached.
func tileHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var chunks = mux.Vars(r)
	if chunks == nil {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid, _ = strconv.ParseUint(chunks["aid"], 10, 64)

	var prf *Profile
	if prf = prflist.ByID(ID_t(aid)); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECtilenoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var puid Puid_t
	if err = puid.Set(chunks["puid"]); err != nil {
		WriteError400(w, r, err, AECtilenopuid)
		return
	}
	var wdh, _ = strconv.Atoi(chunks["wdh"])
	var hgt, _ = strconv.Atoi(chunks["hgt"])
	if wdh == 0 || hgt == 0 {
		WriteError400(w, r, ErrArgNoRes, AECtilebadres)
		return
	}

	var syspath, ok = syspathcache.Path(puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECtilenopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECtilehidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECtilenoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, r, http.StatusUnsupportedMediaType, ErrNotFile, AECtilenofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECtileaccess)
		return
	}

	var md *MediaData
	if md, err = GetCachedTile(syspath, wdh, hgt); err != nil {
		WriteError500(w, r, err, AECtilebadcnt)
		return
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, puid.String(), starttime, bytes.NewReader(md.Data))
}

// APIHANDLER
func pingAPI(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	WriteStdHeader(w)
	w.Write(body)
}

// APIHANDLER
func reloadAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	_ = auth
	var err error
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		RecNumber int64 `json:"recnumber" yaml:"recnumber" xml:"recnumber"`
		DataSize  int64 `json:"datasize" yaml:"datasize" xml:"datasize"`
	}

	if packager, err = openpackage(); err != nil {
		WriteError500(w, r, err, AECreloadload)
		return
	}
	if err = loadtemplates(); err != nil {
		WriteError500(w, r, err, AECreloadtmpl)
		return
	}

	packager.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		ret.RecNumber++
		return true
	})
	if ts, ok := packager.Tagset(""); ok {
		ret.DataSize = ts.Size()
	}
	WriteOK(w, r, &ret)
}

// APIHANDLER
func srvinfAPI(w http.ResponseWriter, r *http.Request) {
	var ret = XmlMap{
		"started":  UnixJS(starttime),
		"govers":   runtime.Version(),
		"os":       runtime.GOOS,
		"numcpu":   runtime.NumCPU(),
		"maxprocs": runtime.GOMAXPROCS(0),
		"exepath":  path.Dir(ToSlash(os.Args[0])),
		"cfgpath":  ConfigPath,
		"wpkpath":  PackPath,
		"cchpath":  CachePath,
	}

	WriteOK(w, r, ret)
}

// APIHANDLER
func memusgAPI(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var ret = XmlMap{
		"buildvers":     buildvers,
		"builddate":     builddate,
		"buildtime":     buildtime,
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
	syspathcache.mux.RLock()
	var pathnum = len(syspathcache.keypath)
	syspathcache.mux.RUnlock()

	var propnum = propcache.Len(false)

	type stat struct {
		size1 float64
		size2 float64
		num   int
	}
	var med stat
	var jpg, png, gif stat
	thumbpkg.Enum(func(fkey string, ts *wpk.Tagset_t) bool {
		var l = float64(ts.Size())
		med.size1 += l
		med.size2 += l * l
		med.num++
		if str, ok := ts.String(wpk.TIDmime); ok {
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
		"pathcchnum":  pathnum,
		"propcchnum":  propnum,
		"tmbcchnum":   gif.num + png.num + jpg.num,
		"tmbcchsize1": gif.size1 + png.size1 + jpg.size1,
		"tmbcchsize2": gif.size2 + png.size2 + jpg.size2,
		"tmbjpgnum":   jpg.num,
		"tmbjpgsize1": jpg.size1,
		"tmbjpgsize2": jpg.size2,
		"tmbpngnum":   png.num,
		"tmbpngsize1": png.size1,
		"tmbpngsize2": png.size2,
		"tmbgifnum":   gif.num,
		"tmbgifsize1": gif.size1,
		"tmbgifsize2": gif.size2,
		"medcchnum":   med.num,
		"medcchsize1": med.size1,
		"medcchsize2": med.size2,
	}

	WriteOK(w, r, ret)
}

// APIHANDLER
func getlogAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []LogItem `json:"list" yaml:"list" xml:"list"`
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

	ret.List = make([]LogItem, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		ret.List[i] = h.Value.(LogItem)
		h = h.Prev()
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ishomeAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID ID_t `json:"aid" yaml:"aid" xml:"aid,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		IsHome bool `json:"ishome" yaml:"ishome" xml:"ishome"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECishomenoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	if auth == prf {
		ret.IsHome = true
	} else if prf.IsShared(CPhome) {
		for _, fpath := range CatKeyPath {
			if fpath == CPhome {
				continue
			}
			if prf.IsShared(fpath) {
				if _, err := propcache.Get(fpath); err == nil {
					ret.IsHome = true
					break
				}
			}
		}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ctgrAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Pather `json:"list" yaml:"list" xml:"list>prop"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECctgrnodata)
		return
	}

	var catpath, ok = CatKeyPath[arg.PUID]
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECctgrnopath)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECctgrnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	if auth != prf && !prf.IsShared(catpath) {
		WriteError(w, r, http.StatusForbidden, ErrNotShared, AECctgrnoshr)
		return
	}
	var catprop = func(puids []Puid_t) {
		for _, puid := range puids {
			if fpath, ok := syspathcache.Path(puid); ok {
				if prop, err := propcache.Get(fpath); err == nil {
					ret.List = append(ret.List, prop.(Pather))
				}
			}
		}
	}
	switch arg.PUID {
	case PUIDhome:
		for puid := Puid_t(1); puid < PUIDreserved; puid++ {
			if puid == PUIDhome {
				continue
			}
			if fpath, ok := CatKeyPath[puid]; ok {
				if auth == prf || prf.IsShared(fpath) {
					if prop, err := propcache.Get(fpath); err == nil {
						ret.List = append(ret.List, prop.(Pather))
					}
				}
			}
		}
	case PUIDdrives:
		ret.List = prf.ScanRoots()
	case PUIDshares:
		ret.List = prf.ScanShares()
	case PUIDmedia:
		catprop(dircache.Categories([]int{FGvideo, FGaudio, FGimage}, 0.5))
	case PUIDvideo:
		catprop(dircache.Category(FGvideo, 0.5))
	case PUIDaudio:
		catprop(dircache.Category(FGaudio, 0.5))
	case PUIDimage:
		catprop(dircache.Category(FGimage, 0.5))
	case PUIDbooks:
		catprop(dircache.Category(FGbooks, 0.5))
	case PUIDtexts:
		catprop(dircache.Category(FGtexts, 0.5))
	default:
		WriteError(w, r, http.StatusMethodNotAllowed, ErrNotCat, AECctgrnotcat)
		return
	}

	Log.Infof("id%d: navigate to %s", prf.ID, catpath)
	usermsg <- UsrMsg{r, "path", arg.PUID}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func folderAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid,omitempty" yaml:"puid,omitempty" xml:"puid,omitempty"`
		Path string `json:"path,omitempty" yaml:"path,omitempty" xml:"path,omitempty"`
		Ext  string `json:"ext,omitempty" yaml:"ext,omitempty" xml:"ext,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Pather `json:"list" yaml:"list" xml:"list"`
		Skip int      `json:"skip" yaml:"skip" xml:"skip"`
		PUID Puid_t   `json:"puid" yaml:"puid" xml:"puid"`
		Path string   `json:"path" yaml:"path" xml:"path"`
		Name string   `json:"shrname" yaml:"shrname" xml:"shrname"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 && len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECfoldernodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECfoldernoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath string
	if len(arg.Path) > 0 {
		syspath = path.Clean(ToSlash(arg.Path))
		if syspath[0] == '.' {
			WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECfolderroot)
			return
		}
		syspath = UnfoldPath(syspath)
		ret.PUID = syspathcache.Cache(syspath)
	} else {
		var ok bool
		if syspath, ok = syspathcache.Path(arg.PUID); !ok {
			WriteError(w, r, http.StatusNotFound, ErrNoPath, AECfoldernopath)
			return
		}
		ret.PUID = arg.PUID
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECfolderhidden)
		return
	}

	var shrpath, base, cg = prf.GetSharePath(syspath, auth == prf)
	if cg.IsZero() {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECfolderaccess)
		return
	}
	ret.Path = shrpath
	ret.Name = PathBase(base)

	var fi fs.FileInfo
	if fi, err = StatFile(syspath); err != nil {
		WriteError500(w, r, err, AECfolderstat)
		return
	}

	var ext = arg.Ext
	if ext == "" {
		ext = GetFileExt(syspath)
	}

	var t = time.Now()
	if !fi.IsDir() && IsTypePlaylist(ext) {
		var file io.ReadCloser
		if file, err = OpenFile(syspath); err != nil {
			WriteError500(w, r, err, AECfolderopen)
			return
		}
		defer file.Close()

		var pl Playlist
		pl.Dest = path.Dir(syspath)
		switch ext {
		case ".m3u", ".m3u8":
			if _, err = pl.ReadM3U(file); err != nil {
				WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderm3u)
				return
			}
		case ".wpl":
			if _, err = pl.ReadWPL(file); err != nil {
				WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderwpl)
				return
			}
		case ".pls":
			if _, err = pl.ReadPLS(file); err != nil {
				WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderpls)
				return
			}
		case ".asx":
			if _, err = pl.ReadASX(file); err != nil {
				WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderasx)
				return
			}
		case ".xspf":
			if _, err = pl.ReadXSPF(file); err != nil {
				WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderxspf)
				return
			}
		default:
			WriteError(w, r, http.StatusUnsupportedMediaType, ErrNotPlay, AECfolderformat)
			return
		}

		var prop interface{}
		for _, track := range pl.Tracks {
			var fpath = ToSlash(track.Location)
			if !prf.IsHidden(fpath) {
				var cg = prf.PathAccess(fpath, auth == prf)
				var grp = GetFileGroup(fpath)
				if cg[grp] {
					if prop, err = propcache.Get(fpath); err == nil {
						ret.List = append(ret.List, prop.(Pather))
						continue
					}
				}
			}
		}
		ret.Skip = len(pl.Tracks) - len(ret.List)
	} else {
		if ret.List, ret.Skip, err = ScanDir(syspath, &cg, func(fpath string) bool {
			return !prf.IsHidden(fpath)
		}); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusNotFound, err, AECfolderabsent)
			} else {
				WriteError500(w, r, err, AECfolderfail)
			}
			return
		}
	}

	Log.Infof("id%d: navigate to %s, items %d, timeout %s", prf.ID, syspath, len(ret.List), time.Since(t))
	usermsg <- UsrMsg{r, "path", ret.PUID}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func ispathAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		Path string `json:"path" yaml:"path" xml:"path"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECispathnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECispathdeny)
		return
	}

	var syspath = path.Clean(ToSlash(arg.Path))
	if syspath[0] == '.' {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECispathroot)
		return
	}
	syspath = UnfoldPath(syspath)

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECispathhidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		var ptr *FileProp
		prop = ptr // write "null" as reply
	}
	WriteOK(w, r, prop)
}

// APIHANDLER
func shraddAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Shared bool `json:"shared" yaml:"shared" xml:"shared"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshraddnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECshraddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECshradddeny)
		return
	}

	var syspath, ok = syspathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECshraddnopath)
	}
	if !prf.PathAdmin(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECshraddaccess)
		return
	}

	ret.Shared = prf.AddShare(syspath)
	Log.Infof("id%d: add share '%s' as %s", prf.ID, syspath, arg.PUID)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func shrdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshrdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
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

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		Path string `json:"path" yaml:"path" xml:"path"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvaddnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvaddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvadddeny)
		return
	}

	var syspath = path.Clean(ToSlash(arg.Path))
	if syspath[0] == '.' {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECdrvaddroot)
		return
	}
	// append slash to disk root to prevent open current dir on this disk
	if strings.HasSuffix(syspath, ":") {
		syspath += "/"
	}
	syspath = UnfoldPath(syspath)
	if prf.RootIndex(syspath) >= 0 {
		WriteOK(w, r, nil)
		return
	}

	var dk DriveKit
	dk.Setup(syspath)
	if err = dk.Scan(syspath); err != nil {
		WriteError400(w, r, err, AECdrvaddfile)
		return
	}

	prf.mux.Lock()
	prf.Roots = append(prf.Roots, syspath)
	prf.mux.Unlock()

	WriteOK(w, r, dk)
}

// APIHANDLER
func drvdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvdeldeny)
		return
	}

	var syspath, ok = syspathcache.Path(arg.PUID)
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

// APIHANDLER
func edtcopyAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		Src Puid_t `json:"src" yaml:"src" xml:"src"`
		Dst Puid_t `json:"dst" yaml:"dst" xml:"dst"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == 0 || arg.Dst == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECedtcopynodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtcopynoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtcopydeny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	var ok bool
	if srcpath, ok = syspathcache.Path(arg.Src); !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECedtcopynopath)
		return
	}
	if dstpath, ok = syspathcache.Path(arg.Dst); !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECedtcopynodest)
		return
	}
	dstpath = path.Join(dstpath, path.Base(srcpath))

	var prop Pather
	// copies file or dir from source to destination
	var filecopy func(srcpath, dstpath string) (err error)
	filecopy = func(srcpath, dstpath string) (err error) {
		// generate unique destination filename
		if !arg.Ovw {
			var ext = path.Ext(dstpath)
			var org = dstpath[:len(dstpath)-len(ext)]
			var i = 1
			for {
				if _, err = os.Stat(dstpath); errors.Is(err, fs.ErrNotExist) {
					break
				}
				i++
				dstpath = fmt.Sprintf("%s (%d)%s", org, i, ext)
				if i > 100 {
					err = ErrFileOver
					WriteError500(w, r, err, AECedtcopyover)
					return
				}
			}
		}

		var src, dst *os.File
		var fi fs.FileInfo
		// open source file
		if src, err = os.Open(srcpath); err != nil {
			WriteError500(w, r, err, AECedtcopyopsrc)
			return
		}
		defer func() {
			src.Close()
			if fi != nil {
				os.Chtimes(dstpath, fi.ModTime(), fi.ModTime())
			}
		}()

		if fi, err = src.Stat(); err != nil {
			WriteError500(w, r, err, AECedtcopystatsrc)
			return
		}
		if fi.IsDir() {
			// create destination dir
			if err = os.Mkdir(dstpath, 0644); err != nil && !errors.Is(err, fs.ErrExist) {
				WriteError500(w, r, err, AECedtcopymkdir)
				return
			}

			// get returned dir properties now
			if prop == nil {
				prop = MakeProp(dstpath, fi)
			}

			// copy dir content
			var files []fs.DirEntry
			if files, err = src.ReadDir(-1); err != nil {
				WriteError500(w, r, err, AECedtcopyrd)
				return
			}
			for _, file := range files {
				var name = file.Name()
				if err = filecopy(path.Join(srcpath, name), path.Join(dstpath, name)); err != nil {
					return // error already written
				}
			}
		} else {
			// create destination file
			if dst, err = os.Create(dstpath); err != nil {
				WriteError500(w, r, err, AECedtcopyopdst)
				return
			}
			defer dst.Close()

			// copy file content
			if _, err = io.Copy(dst, src); err != nil {
				WriteError500(w, r, err, AECedtcopycopy)
				return
			}

			// get returned file properties at last
			if prop == nil {
				var fi fs.FileInfo
				if fi, err = dst.Stat(); err != nil {
					WriteError500(w, r, err, AECedtcopystatfile)
					return
				}
				prop = MakeProp(dstpath, fi)
			}
		}
		return
	}
	if err = filecopy(srcpath, dstpath); err != nil {
		return // error already written
	}

	WriteOK(w, r, prop)
}

// APIHANDLER
func edtrenameAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		Src Puid_t `json:"src" yaml:"src" xml:"src"`
		Dst Puid_t `json:"dst" yaml:"dst" xml:"dst"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == 0 || arg.Dst == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECedtrennodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtrennoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtrendeny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	var ok bool
	if srcpath, ok = syspathcache.Path(arg.Src); !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECedtrennopath)
		return
	}
	if dstpath, ok = syspathcache.Path(arg.Dst); !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECedtrennodest)
		return
	}
	dstpath = path.Join(dstpath, path.Base(srcpath))

	// generate unique destination filename
	if !arg.Ovw {
		var ext = path.Ext(dstpath)
		var org = dstpath[:len(dstpath)-len(ext)]
		var i = 1
		for {
			if _, err = os.Stat(dstpath); errors.Is(err, fs.ErrNotExist) {
				break
			}
			i++
			dstpath = fmt.Sprintf("%s (%d)%s", org, i, ext)
			if i > 100 {
				err = ErrFileOver
				WriteError500(w, r, err, AECedtrenover)
				return
			}
		}
	}

	// rename destination file
	if err = os.Rename(srcpath, dstpath); err != nil && !errors.Is(err, fs.ErrExist) {
		WriteError500(w, r, err, AECedtrenmove)
		return
	}

	// get returned file properties at last
	var fi fs.FileInfo
	if fi, err = os.Stat(dstpath); err != nil {
		WriteError500(w, r, err, AECedtrenstat)
		return
	}
	var prop = MakeProp(dstpath, fi)

	WriteOK(w, r, prop)
}

// APIHANDLER
func edtdeleteAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECedtdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtdeldeny)
		return
	}

	var syspath, ok = syspathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECedtdelnopath)
	}

	if err = os.RemoveAll(syspath); err != nil {
		WriteError500(w, r, err, AECedtdelremove)
		return
	}

	WriteOK(w, r, nil)
}

const rangepool = 4

// Haversine uses formula to calculate the great-circle distance between
// two points – that is, the shortest distance over the earth’s surface –
// giving an ‘as-the-crow-flies’ distance between the points (ignoring
// any hills they fly over, of course!).
//
// See https://www.movable-type.co.uk/scripts/latlong.html
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // metres
	const πrad = math.Pi / 180
	var (
		φ1 = lat1 * πrad // φ, λ in radians
		φ2 = lat2 * πrad
		Δφ = (lat2 - lat1) * πrad
		Δλ = (lon2 - lon1) * πrad
		a  = math.Sin(Δφ/2)*math.Sin(Δφ/2) +
			math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
		c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		d = R * c // in metres
	)
	return d
}

// APIHANDLER
func gpsrangeAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Latitude  float64 `json:"lat" yaml:"lat" xml:"lat"`
		Longitude float64 `json:"lon" yaml:"lon" xml:"lon"`
		Radius    float64 `json:"radius" yaml:"radius" xml:"radius"` // in meters
		Limit     int     `json:"limit,omitempty" yaml:"limit,omitempty" xml:"limit,omitempty"`
		Time1     unix_t  `json:"time1,omitempty" yaml:"time1,omitempty" xml:"time1,omitempty"`
		Time2     unix_t  `json:"time2,omitempty" yaml:"time2,omitempty" xml:"time2,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Puid_t `json:"list" yaml:"list" xml:"list"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Latitude == 0 || arg.Longitude == 0 || arg.Radius == 0 {
		WriteError400(w, r, ErrNoData, AECgpsrangenodata)
		return
	}

	gpscache.Range(func(key interface{}, value interface{}) bool {
		var gps = value.(*GpsInfo)
		var d = Haversine(arg.Latitude, arg.Longitude, gps.Latitude, gps.Longitude)
		if d > arg.Radius {
			return true
		}
		if arg.Time1 > 0 && gps.DateTime < arg.Time1 {
			return true
		}
		if arg.Time2 > 0 && gps.DateTime > arg.Time2 {
			return true
		}
		ret.List = append(ret.List, key.(Puid_t))
		return arg.Limit > 0 && len(ret.List) < arg.Limit
	})

	WriteOK(w, r, &ret)
}

// The End.
