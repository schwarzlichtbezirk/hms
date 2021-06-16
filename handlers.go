package hms

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AECnull    = 0
	AECbadbody = 1
	AECnoreq   = 2
	AECbadjson = 3

	// auth

	AECnoauth     = 4
	AECtokenless  = 5
	AECtokenerror = 6
	AECtokenbad   = 7
	AECtokennoacc = 8

	// page

	AECpageabsent = 10
	AECfileabsent = 11

	// file

	AECfilebadaccid = 12
	AECfilenoacc    = 13
	AECfilehroot    = 14
	AECfilehidden   = 15
	AECfilenoprop   = 16
	AECfilenofile   = 17
	AECfileaccess   = 18

	// media

	AECmediabadaccid = 20
	AECmedianoacc    = 21
	AECmedianopath   = 22
	AECmediahidden   = 23
	AECmedianoprop   = 24
	AECmedianofile   = 25
	AECmediaaccess   = 26
	AECmediaabsent   = 27
	AECmediaopen     = 28
	AECmediabadcnt   = 29

	// thumb

	AECthumbbadaccid = 30
	AECthumbnoacc    = 31
	AECthumbnopath   = 32
	AECthumbhidden   = 33
	AECthumbnoprop   = 34
	AECthumbnofile   = 35
	AECthumbaccess   = 36
	AECthumbabsent   = 37
	AECthumbbadcnt   = 38

	// pubkey

	AECpubkeyrand = 40

	// signin

	AECsigninnodata = 41
	AECsigninnoacc  = 42
	AECsigninpkey   = 43
	AECsignindeny   = 44

	// refrsh

	AECrefrshnodata = 45
	AECrefrshparse  = 46

	// reload

	AECreloadload = 50
	AECreloadtmpl = 51

	// getlog

	AECgetlogbadnum = 52

	// ishome

	AECishomenoacc = 53

	// ctgr

	AECctgrnodata = 60
	AECctgrnopath = 61
	AECctgrnocid  = 62
	AECctgrnoacc  = 63
	AECctgrnoshr  = 64
	AECctgrnotcat = 65

	// folder

	AECfoldernodata = 70
	AECfoldernoacc  = 71
	AECfolderroot   = 72
	AECfoldernopath = 73
	AECfolderhidden = 74
	AECfolderaccess = 75
	AECfolderabsent = 76
	AECfolderfail   = 77

	// playlist

	AECplaylistnodata = 80
	AECplaylistnoacc  = 81
	AECplaylistnopath = 82
	AECplaylisthidden = 83
	AECplaylistaccess = 84
	AECplaylistopen   = 85
	AECplaylistm3u    = 86
	AECplaylistwpl    = 87
	AECplaylistpls    = 88
	AECplaylistasx    = 89
	AECplaylistxspf   = 90
	AECplaylistformat = 91

	// ispath

	AECispathnoacc  = 92
	AECispathdeny   = 93
	AECispathroot   = 94
	AECispathhidden = 95

	// tmb/chk

	AECtmbchknodata = 96

	// tmb/scn

	AECtmbscnnodata = 97
	AECtmbscnnoacc  = 98

	// share/lst

	AECshrlstnoacc = 100
	AECshrlstnoshr = 101

	// share/add

	AECshraddnodata = 102
	AECshraddnoacc  = 103
	AECshradddeny   = 104
	AECshraddnopath = 105
	AECshraddaccess = 106

	// share/del

	AECshrdelnodata = 107
	AECshrdelnoacc  = 108
	AECshrdeldeny   = 109

	// drive/lst

	AECdrvlstnoacc = 110
	AECdrvlstnoshr = 111

	// drive/add

	AECdrvaddnodata = 112
	AECdrvaddnoacc  = 113
	AECdrvadddeny   = 114
	AECdrvaddroot   = 115
	AECdrvaddfile   = 116

	// drive/del

	AECdrvdelnodata = 117
	AECdrvdelnoacc  = 118
	AECdrvdeldeny   = 119
	AECdrvdelnopath = 120
)

// HTTP error messages
var (
	ErrNoJSON = errors.New("data not given")
	ErrNoData = errors.New("data is empty")

	ErrNotFound  = errors.New("404 page not found")
	ErrArgNoNum  = errors.New("'num' parameter not recognized")
	ErrArgNoPath = errors.New("'path' argument required")
	ErrArgNoHash = errors.New("'puid' or 'path' argument required")
	ErrNotDir    = errors.New("path is not directory")
	ErrNoPath    = errors.New("path is not found")
	ErrDeny      = errors.New("access denied for specified authorization")
	ErrNotShared = errors.New("access to specified resource does not shared")
	ErrHidden    = errors.New("access to specified file path is disabled")
	ErrNoAccess  = errors.New("profile has no access to specified file path")
	ErrNotCat    = errors.New("only categories can be accepted")
	ErrNotPlay   = errors.New("file can not be read as playlist")
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
			WriteError(w, http.StatusNotFound, ErrNotFound, AECpageabsent)
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
						aid = int(u64)
					}
				}
				usermsg <- UsrMsg{r, "page", aid}
			}()
		}

		WriteHTMLHeader(w)
		http.ServeContent(w, r, alias, starttime, bytes.NewReader(content))
	}
}

// APIHANDLER
func fileHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid uint64
	if aid, err = strconv.ParseUint(chunks[1][2:], 10, 32); err != nil {
		WriteError400(w, err, AECfilebadaccid)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(int(aid)); prf == nil {
		WriteError400(w, ErrNoAcc, AECfilenoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	var syspath = path.Clean(strings.Join(chunks[3:], "/"))
	if syspath[0] == '.' {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECfilehroot)
		return
	}
	syspath = UnfoldPath(syspath)

	if strings.HasPrefix(syspath, "http://") || strings.HasPrefix(syspath, "https://") {
		http.Redirect(w, r, syspath, http.StatusMovedPermanently)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECfilehidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, AECfilenoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, http.StatusUnsupportedMediaType, ErrNotFile, AECfilenofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECfileaccess)
		return
	}

	go func() {
		if _, ok := r.Header["If-Range"]; !ok {
			// not partial content
			usermsg <- UsrMsg{r, "file", fp.PUID()}
			Log.Printf("id%d: serve %s", prf.ID, PathBase(syspath))
		} else {
			// update statistics for partial content
			userajax <- r
		}
	}()

	var content io.ReadSeekCloser // VFile
	if content, err = OpenFile(syspath); err != nil {
		WriteError500(w, err, AECmediaopen)
		return
	}
	defer content.Close()

	WriteStdHeader(w)
	http.ServeContent(w, r, syspath, TimeJS(fp.Time()), content)
}

// Hands out converted media files if them can be cached.
func mediaHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid uint64
	if aid, err = strconv.ParseUint(chunks[1][2:], 10, 32); err != nil {
		WriteError400(w, err, AECmediabadaccid)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(int(aid)); prf == nil {
		WriteError400(w, ErrNoAcc, AECmedianoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, AECmedianopath)
		return
	}

	if strings.HasPrefix(syspath, "http://") || strings.HasPrefix(syspath, "https://") {
		http.Redirect(w, r, syspath, http.StatusMovedPermanently)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECmediahidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, AECmedianoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, http.StatusUnsupportedMediaType, ErrNotFile, AECmedianofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECmediaaccess)
		return
	}

	var val interface{}
	if val, err = mediacache.Get(puid); err != nil {
		if !errors.Is(err, ErrUncacheable) {
			WriteError(w, http.StatusNotFound, err, AECmediaabsent)
			return
		}

		go func() {
			if _, ok := r.Header["If-Range"]; !ok {
				// not partial content
				usermsg <- UsrMsg{r, "file", puid}
				Log.Printf("id%d: serve %s", prf.ID, PathBase(syspath))
			} else {
				// update statistics for partial content
				userajax <- r
			}
		}()

		var content io.ReadSeekCloser // VFile
		if content, err = OpenFile(syspath); err != nil {
			WriteError500(w, err, AECmediaopen)
			return
		}
		defer content.Close()

		WriteStdHeader(w)
		http.ServeContent(w, r, syspath, TimeJS(fp.Time()), content)
		return
	}
	var md *MediaData
	if md, ok = val.(*MediaData); !ok || md == nil {
		WriteError500(w, ErrBadMedia, AECmediabadcnt)
		return
	}

	go func() {
		if _, ok := r.Header["If-Range"]; !ok {
			// not partial content
			usermsg <- UsrMsg{r, "file", puid}
			Log.Printf("id%d: media %s", prf.ID, PathBase(syspath))
		} else {
			// update statistics for partial content
			userajax <- r
		}
	}()
	w.Header().Set("Content-Type", md.Mime)
	http.ServeContent(w, r, puid, starttime, bytes.NewReader(md.Data))
}

// Hands out thumbnails for given files if them cached.
func thumbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid uint64
	if aid, err = strconv.ParseUint(chunks[1][2:], 10, 32); err != nil {
		WriteError400(w, err, AECthumbbadaccid)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(int(aid)); prf == nil {
		WriteError400(w, ErrNoAcc, AECthumbnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, AECthumbnopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECthumbhidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, AECthumbnoprop)
		return
	}
	var fp = prop.(Pather)
	if fp.Type() != FTfile {
		WriteError(w, http.StatusUnsupportedMediaType, ErrNotFile, AECthumbnofile)
		return
	}
	var cg = prf.PathAccess(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECthumbaccess)
		return
	}

	var val interface{}
	if val, err = thumbcache.Get(puid); err != nil {
		WriteError(w, http.StatusNotFound, err, AECthumbabsent)
		return
	}
	var md *MediaData
	if md, ok = val.(*MediaData); !ok || md == nil {
		WriteError500(w, ErrBadMedia, AECthumbbadcnt)
		return
	}
	w.Header().Set("Content-Type", md.Mime)
	http.ServeContent(w, r, puid, starttime, bytes.NewReader(md.Data))
}

// APIHANDLER
func pingAPI(w http.ResponseWriter, r *http.Request) {
	var body, _ = io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	WriteJSONHeader(w)
	w.Write(body)
}

// APIHANDLER
func purgeAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	propcache.Purge()
	thumbcache.Purge()

	prflist.mux.RLock()
	for _, prf := range prflist.list {
		prf.UpdateShares()
	}
	prflist.mux.RUnlock()

	WriteOK(w, nil)
}

// APIHANDLER
func reloadAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var ret struct {
		RecNumber int   `json:"recnumber"`
		DataSize  int64 `json:"datasize"`
	}

	if packager, err = openimage(); err != nil {
		WriteError500(w, err, AECreloadload)
		return
	}
	if err = loadtemplates(); err != nil {
		WriteError500(w, err, AECreloadtmpl)
		return
	}

	ret.RecNumber = packager.RecNumber()
	ret.DataSize = packager.DataSize()
	WriteOK(w, &ret)
}

// APIHANDLER
func srvinfAPI(w http.ResponseWriter, r *http.Request) {
	var ret = map[string]interface{}{
		"started":  UnixJS(starttime),
		"govers":   runtime.Version(),
		"os":       runtime.GOOS,
		"numcpu":   runtime.NumCPU(),
		"maxprocs": runtime.GOMAXPROCS(0),
		"destpath": destpath,
		"confpath": confpath,
	}

	WriteOK(w, ret)
}

// APIHANDLER
func memusgAPI(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var ret = map[string]interface{}{
		"running":       time.Since(starttime) / time.Millisecond,
		"heapalloc":     mem.HeapAlloc,
		"heapsys":       mem.HeapSys,
		"totalalloc":    mem.TotalAlloc,
		"nextgc":        mem.NextGC,
		"numgc":         mem.NumGC,
		"pausetotalns":  mem.PauseTotalNs,
		"gccpufraction": mem.GCCPUFraction,
	}

	WriteOK(w, ret)
}

// APIHANDLER
func cchinfAPI(w http.ResponseWriter, r *http.Request) {
	pathcache.mux.RLock()
	var pathnum = len(pathcache.keypath)
	pathcache.mux.RUnlock()

	var propnum = propcache.Len(false)

	var tc = thumbcache.GetALL(false)
	type stat struct {
		size1 float64
		size2 float64
		num   int
	}
	var jpg, png, gif stat
	for _, v := range tc {
		var md = v.(*MediaData)
		var s *stat
		switch md.Mime {
		case "image/gif":
			s = &gif
		case "image/png":
			s = &png
		case "image/jpeg":
			s = &jpg
		default:
			panic("unexpected MIME type in cache " + md.Mime)
		}
		var l = float64(len(md.Data))
		s.size1 += l
		s.size2 += l * l
		s.num++
	}

	var mc = mediacache.GetALL(false)
	var med stat
	for _, v := range mc {
		var md = v.(*MediaData)
		var l = float64(len(md.Data))
		med.size1 += l
		med.size2 += l * l
		med.num++
	}

	var ret = map[string]interface{}{
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

	WriteOK(w, ret)
}

// APIHANDLER
func getlogAPI(w http.ResponseWriter, r *http.Request) {
	var err error

	var size = Log.Size()

	// get arguments
	var num int
	if s := r.FormValue("num"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, ErrArgNoNum, AECgetlogbadnum)
			return
		}
		num = int(i64)
	}
	if num <= 0 || num > size {
		num = size
	}

	var ret = make([]interface{}, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		ret[i] = h.Value
		h = h.Prev()
	}

	WriteOK(w, ret)
}

// APIHANDLER
func ishomeAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret bool

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECishomenoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	if auth == prf {
		ret = true
	} else if prf.IsShared(CPhome) {
		for _, fpath := range CatPath {
			if fpath == CPhome {
				continue
			}
			if prf.IsShared(fpath) {
				if _, err := propcache.Get(fpath); err == nil {
					ret = true
					break
				}
			}
		}
	}

	Log.Printf("id%d: navigate to home", prf.ID)
	WriteOK(w, ret)
}

// APIHANDLER
func ctgrAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
		CID  string `json:"cid"`
	}
	var ret = []Pather{}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	var catpath string
	if len(arg.CID) > 0 {
		var ok bool
		if catpath, ok = CidCatPath[arg.CID]; !ok {
			WriteError400(w, ErrArgNoPath, AECctgrnocid)
			return
		}
		arg.PUID = pathcache.Cache(catpath)
	} else if len(arg.PUID) > 0 {
		var ok bool
		if catpath, ok = pathcache.Path(arg.PUID); !ok {
			WriteError(w, http.StatusNotFound, ErrNoPath, AECctgrnopath)
			return
		}
	} else {
		WriteError400(w, ErrArgNoPath, AECctgrnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECctgrnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	if auth != prf && !prf.IsShared(catpath) {
		WriteError(w, http.StatusForbidden, ErrNotShared, AECctgrnoshr)
		return
	}
	var catprop = func(puids []string) {
		for _, puid := range puids {
			if fpath, ok := pathcache.Path(puid); ok {
				if prop, err := propcache.Get(fpath); err == nil {
					ret = append(ret, prop.(Pather))
				}
			}
		}
	}
	switch catpath {
	case CPhome:
		for _, fpath := range CatPath {
			if fpath == CPhome {
				continue
			}
			if auth == prf || prf.IsShared(fpath) {
				if prop, err := propcache.Get(fpath); err == nil {
					ret = append(ret, prop.(Pather))
				}
			}
		}
	case CPdrives:
		ret = prf.ScanRoots()
	case CPshares:
		ret = prf.ScanShares()
	case CPmedia:
		catprop(dircache.Categories([]int{FGvideo, FGaudio, FGimage}, 0.5))
	case CPvideo:
		catprop(dircache.Category(FGvideo, 0.5))
	case CPaudio:
		catprop(dircache.Category(FGaudio, 0.5))
	case CPimage:
		catprop(dircache.Category(FGimage, 0.5))
	case CPbooks:
		catprop(dircache.Category(FGbooks, 0.5))
	case CPtexts:
		catprop(dircache.Category(FGtexts, 0.5))
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrNotCat, AECctgrnotcat)
		return
	}

	usermsg <- UsrMsg{r, "path", arg.PUID}
	Log.Printf("id%d: navigate to %s", prf.ID, catpath)
	WriteOK(w, ret)
}

// APIHANDLER
func folderAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid,omitempty"`
		Path string `json:"path,omitempty"`
	}
	var ret struct {
		List []Pather `json:"list"`
		PUID string   `json:"puid"`
		Path string   `json:"path"`
		Name string   `json:"shrname"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUID) == 0 && len(arg.Path) == 0 {
		WriteError400(w, ErrArgNoPath, AECfoldernodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECfoldernoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	var syspath string
	if len(arg.Path) > 0 {
		syspath = path.Clean(ToSlash(arg.Path))
		if syspath[0] == '.' {
			WriteError(w, http.StatusForbidden, ErrNoAccess, AECfolderroot)
			return
		}
		syspath = UnfoldPath(syspath)
		ret.PUID = pathcache.Cache(syspath)
	} else {
		var ok bool
		if syspath, ok = pathcache.Path(arg.PUID); !ok {
			WriteError(w, http.StatusNotFound, ErrNoPath, AECfoldernopath)
			return
		}
		ret.PUID = arg.PUID
	}

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECfolderhidden)
		return
	}

	var shrpath, base, cg = prf.GetSharePath(syspath, auth == prf)
	if cg.IsZero() {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECfolderaccess)
		return
	}
	ret.Path = shrpath
	ret.Name = PathBase(base)

	if ret.List, err = ScanDir(syspath, &cg, func(fpath string) bool {
		return prf.IsHidden(fpath)
	}); err != nil {
		if os.IsNotExist(err) {
			WriteError(w, http.StatusNotFound, err, AECfolderabsent)
		} else {
			WriteError500(w, err, AECfolderfail)
		}
		return
	}
	usermsg <- UsrMsg{r, "path", ret.PUID}
	Log.Printf("id%d: navigate to %s", prf.ID, syspath)

	WriteOK(w, ret)
}

// APIHANDLER
func playlistAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid,omitempty"`
		Ext  string `json:"ext,omitempty"`
	}
	var ret struct {
		List []Pather `json:"list"`
		Skip int      `json:"skip"`
		Path string   `json:"path"`
		Name string   `json:"shrname"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUID) == 0 {
		WriteError400(w, ErrArgNoPath, AECplaylistnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECplaylistnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	var syspath string
	var ok bool
	if syspath, ok = pathcache.Path(arg.PUID); !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, AECplaylistnopath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECplaylisthidden)
		return
	}

	var shrpath, base, cg = prf.GetSharePath(syspath, auth == prf)
	var grp = GetFileGroup(syspath)
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECplaylistaccess)
		return
	}
	ret.Path = shrpath
	ret.Name = PathBase(base)

	var file VFile
	if file, err = OpenFile(syspath); err != nil {
		WriteError500(w, err, AECplaylistopen)
		return
	}
	var pl Playlist
	pl.Dest = path.Dir(syspath)
	var ext = arg.Ext
	if ext == "" {
		ext = GetFileExt(syspath)
	}
	switch ext {
	case ".m3u", ".m3u8":
		if _, err = pl.ReadM3U(file); err != nil {
			WriteError(w, http.StatusUnsupportedMediaType, err, AECplaylistm3u)
			return
		}
	case ".wpl":
		if _, err = pl.ReadWPL(file); err != nil {
			WriteError(w, http.StatusUnsupportedMediaType, err, AECplaylistwpl)
			return
		}
	case ".pls":
		if _, err = pl.ReadPLS(file); err != nil {
			WriteError(w, http.StatusUnsupportedMediaType, err, AECplaylistpls)
			return
		}
	case ".asx":
		if _, err = pl.ReadASX(file); err != nil {
			WriteError(w, http.StatusUnsupportedMediaType, err, AECplaylistasx)
			return
		}
	case ".xspf":
		if _, err = pl.ReadXSPF(file); err != nil {
			WriteError(w, http.StatusUnsupportedMediaType, err, AECplaylistxspf)
			return
		}
	default:
		WriteError(w, http.StatusUnsupportedMediaType, ErrNotPlay, AECplaylistformat)
		return
	}

	var prop interface{}
	for _, track := range pl.Tracks {
		var cg = prf.PathAccess(track.Location, auth == prf)
		var grp = GetFileGroup(track.Location)
		if cg[grp] {
			if prop, err = propcache.Get(track.Location); err == nil {
				ret.List = append(ret.List, prop.(Pather))
				continue
			}
		}
		ret.Skip++
	}

	usermsg <- UsrMsg{r, "path", arg.PUID}
	Log.Printf("id%d: navigate to %s", prf.ID, syspath)

	WriteOK(w, ret)
}

// APIHANDLER
func ispathAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECispathnoacc)
		return
	}
	if auth != prf {
		WriteError(w, http.StatusForbidden, ErrDeny, AECispathdeny)
		return
	}

	var syspath = path.Clean(ToSlash(arg.Path))
	if syspath[0] == '.' {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECispathroot)
		return
	}
	syspath = UnfoldPath(syspath)

	if prf.IsHidden(syspath) {
		WriteError(w, http.StatusForbidden, ErrHidden, AECispathhidden)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		var ptr *FileProp
		prop = ptr // write "null" as reply
	}
	WriteOK(w, prop)
}

// APIHANDLER
func shrlstAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret = []Pather{}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECshrlstnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	if auth != prf && !prf.IsShared(CPshares) {
		WriteError(w, http.StatusForbidden, ErrNotShared, AECshrlstnoshr)
		return
	}

	ret = prf.ScanShares()
	WriteOK(w, ret)
}

// APIHANDLER
func shraddAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUID) == 0 {
		WriteError400(w, ErrArgNoPath, AECshraddnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECshraddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, http.StatusForbidden, ErrDeny, AECshradddeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, AECshraddnopath)
	}
	if !prf.PathAdmin(syspath) {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECshraddaccess)
		return
	}

	var ret = prf.AddShare(syspath)
	Log.Printf("id%d: add share '%s' as %s", prf.ID, syspath, arg.PUID)

	WriteOK(w, ret)
}

// APIHANDLER
func shrdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}
	var ok bool

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUID) == 0 {
		WriteError400(w, ErrArgNoPath, AECshrdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECshrdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, http.StatusForbidden, ErrDeny, AECshrdeldeny)
		return
	}

	if ok = prf.DelShare(arg.PUID); ok {
		Log.Printf("id%d: delete share %s", prf.ID, arg.PUID)
	}

	WriteOK(w, ok)
}

// APIHANDLER
func drvlstAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret []Pather

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECdrvlstnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	if auth != prf && !prf.IsShared(CPdrives) {
		WriteError(w, http.StatusForbidden, ErrNotShared, AECdrvlstnoshr)
		return
	}

	ret = prf.ScanRoots()
	Log.Printf("id%d: navigate to drives", prf.ID)
	WriteOK(w, ret)
}

// APIHANDLER
func drvaddAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, ErrArgNoPath, AECdrvaddnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECdrvaddnoacc)
		return
	}
	if auth != prf {
		WriteError(w, http.StatusForbidden, ErrDeny, AECdrvadddeny)
		return
	}

	var syspath = path.Clean(ToSlash(arg.Path))
	if syspath[0] == '.' {
		WriteError(w, http.StatusForbidden, ErrNoAccess, AECdrvaddroot)
		return
	}
	syspath = UnfoldPath(syspath)
	if prf.RootIndex(syspath) >= 0 {
		WriteOK(w, nil)
		return
	}

	var dk DriveKit
	dk.Setup(syspath)
	if err = dk.Scan(syspath); err != nil {
		WriteError400(w, err, AECdrvaddfile)
		return
	}

	prf.mux.Lock()
	prf.Roots = append(prf.Roots, syspath)
	prf.mux.Unlock()

	WriteOK(w, dk)
}

// APIHANDLER
func drvdelAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUID) == 0 {
		WriteError400(w, ErrArgNoPath, AECdrvdelnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECdrvdelnoacc)
		return
	}
	if auth != prf {
		WriteError(w, http.StatusForbidden, ErrDeny, AECdrvdeldeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, AECdrvdelnopath)
	}

	var i int
	if i = prf.RootIndex(syspath); i >= 0 {
		prf.mux.Lock()
		prf.Roots = append(prf.Roots[:i], prf.Roots[i+1:]...)
		prf.mux.Unlock()
	}

	WriteOK(w, i >= 0)
}

// The End.
