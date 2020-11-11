package hms

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// HTTP error messages
var (
	ErrNoJson = errors.New("data not given")
	ErrNoData = errors.New("data is empty")

	ErrNotFound  = errors.New("404 page not found")
	ErrArgNoNum  = errors.New("'num' parameter not recognized")
	ErrArgNoPath = errors.New("'path' argument required")
	ErrArgNoHash = errors.New("'puid' or 'path' argument required")
	ErrNotDir    = errors.New("path is not directory")
	ErrNoPath    = errors.New("path is not found")
	ErrDeny      = errors.New("access denied for specified authorization")
	ErrNotShared = errors.New("access to specified resource does not shared")
	ErrNoAccess  = errors.New("account has no access to specified file path")
	ErrNotCat    = errors.New("only categories can be accepted")
)

//////////////////////////
// API request handlers //
//////////////////////////

// APIHANDLER
func pageHandler(pref, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var alias = pagealias[name]
		var content, ok = pagecache[pref+alias]
		if !ok {
			WriteError(w, http.StatusNotFound, ErrNotFound, EC_pageabsent)
		}
		userpage <- r

		WriteHtmlHeader(w)
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
		WriteError400(w, err, EC_filebadaccid)
		return
	}

	var acc *Account
	if acc = acclist.ByID(int(aid)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_filenoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	var syspath = UnfoldPath(strings.Join(chunks[3:], "/"))

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_filenoprop)
		return
	}
	var cg = acc.PathAccess(syspath, auth == acc)
	var grp = typetogroup[prop.(Proper).Type()]
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, EC_fileaccess)
		return
	}

	if _, ok := r.Header["If-Range"]; !ok { // not partial content
		userfile <- userfilepath{r, prop.(Proper).PUID()}
		Log.Printf("id%d: serve %s", acc.ID, PathBase(syspath))
	} else {
		userajax <- r // update statistics for partial content
	}
	WriteStdHeader(w)
	http.ServeFile(w, r, syspath)
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
		WriteError400(w, err, EC_mediabadaccid)
		return
	}

	var acc *Account
	if acc = acclist.ByID(int(aid)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_medianoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_medianopath)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_medianoprop)
		return
	}
	var cg = acc.PathAccess(syspath, auth == acc)
	var grp = typetogroup[prop.(Proper).Type()]
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, EC_mediaaccess)
		return
	}

	var val interface{}
	if val, err = mediacache.Get(puid); err != nil {
		if !errors.Is(err, ErrUncacheable) {
			WriteError(w, http.StatusNotFound, err, EC_mediaabsent)
			return
		}

		if _, ok := r.Header["If-Range"]; !ok { // not partial content
			userfile <- userfilepath{r, puid}
			Log.Printf("id%d: serve %s", acc.ID, PathBase(syspath))
		} else {
			userajax <- r // update statistics for partial content
		}
		WriteStdHeader(w)
		http.ServeFile(w, r, syspath)
		return
	}
	var md *MediaData
	if md, ok = val.(*MediaData); !ok || md == nil {
		WriteError500(w, ErrBadMedia, EC_mediabadcnt)
		return
	}

	if _, ok := r.Header["If-Range"]; !ok { // not partial content
		userfile <- userfilepath{r, puid}
		Log.Printf("id%d: media %s", acc.ID, PathBase(syspath))
	} else {
		userajax <- r // update statistics for partial content
	}
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
		WriteError400(w, err, EC_thumbbadaccid)
		return
	}

	var acc *Account
	if acc = acclist.ByID(int(aid)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_thumbnoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_thumbnopath)
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_thumbnoprop)
		return
	}
	var cg = acc.PathAccess(syspath, auth == acc)
	var grp = typetogroup[prop.(Proper).Type()]
	if !cg[grp] {
		WriteError(w, http.StatusForbidden, ErrNoAccess, EC_thumbaccess)
		return
	}

	var val interface{}
	if val, err = thumbcache.Get(puid); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_thumbabsent)
		return
	}
	var md *MediaData
	if md, ok = val.(*MediaData); !ok || md == nil {
		WriteError500(w, ErrBadMedia, EC_thumbbadcnt)
		return
	}
	w.Header().Set("Content-Type", md.Mime)
	http.ServeContent(w, r, puid, starttime, bytes.NewReader(md.Data))
}

// APIHANDLER
func pingApi(w http.ResponseWriter, r *http.Request) {
	var body, _ = ioutil.ReadAll(r.Body)
	WriteOK(w, body)
}

// APIHANDLER
func purgeApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	propcache.Purge()
	thumbcache.Purge()

	acclist.mux.RLock()
	for _, acc := range acclist.list {
		acc.UpdateShares()
	}
	acclist.mux.RUnlock()

	WriteOK(w, nil)
}

// APIHANDLER
func reloadApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error

	if err = packager.OpenWPK(destpath + "hms.wpk"); err != nil {
		WriteError500(w, err, EC_reloadload)
		return
	}
	if err = loadtemplates(); err != nil {
		WriteError500(w, err, EC_reloadtmpl)
		return
	}

	WriteOK(w, &datapack.PackHdr)
}

// APIHANDLER
func srvinfApi(w http.ResponseWriter, r *http.Request) {
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
func memusgApi(w http.ResponseWriter, r *http.Request) {
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
func cchinfApi(w http.ResponseWriter, r *http.Request) {
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
	}

	WriteOK(w, ret)
}

// APIHANDLER
func getlogApi(w http.ResponseWriter, r *http.Request) {
	var err error

	var size = Log.Size()

	// get arguments
	var num int
	if s := r.FormValue("num"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, ErrArgNoNum, EC_getlogbadnum)
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
func homeApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret = []Proper{}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_homebadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_homenoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_homenoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	for _, path := range CatPath {
		if auth == acc || acc.IsShared(path) {
			var prop, _ = propcache.Get(path)
			ret = append(ret, prop.(Proper))
		}
	}

	userpath <- userfilepath{r, ""}
	Log.Printf("id%d: navigate to home", acc.ID)
	WriteOK(w, ret)
}

// APIHANDLER
func ctgrApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
		CID  string `json:"cid"`
	}
	var ret = []Proper{}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_ctgrbadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_ctgrnoreq)
		return
	}

	var catpath string
	if len(arg.CID) > 0 {
		var ok bool
		if catpath, ok = CidCatPath[arg.CID]; !ok {
			WriteError400(w, ErrArgNoPath, EC_ctgrnocid)
			return
		}
		arg.PUID = pathcache.Cache(catpath)
	} else if len(arg.PUID) > 0 {
		var ok bool
		if catpath, ok = pathcache.Path(arg.PUID); !ok {
			WriteError(w, http.StatusNotFound, ErrNoPath, EC_ctgrnopath)
			return
		}
	} else {
		WriteError400(w, ErrArgNoPath, EC_ctgrnodata)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_ctgrnoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	if auth != acc && !acc.IsShared(catpath) {
		WriteError(w, http.StatusForbidden, ErrNotShared, EC_ctgrnoshr)
		return
	}
	var catprop = func(puids []string) {
		for _, puid := range puids {
			if syspath, ok := pathcache.Path(puid); ok {
				if prop, err := propcache.Get(syspath); err == nil {
					ret = append(ret, prop.(Proper))
				}
			}
		}
	}
	switch catpath {
	case CP_drives:
		ret = acc.ScanRoots()
	case CP_shares:
		ret = acc.ScanShares()
	case CP_media:
		catprop(dircache.Categories([]int{FG_video, FG_audio, FG_image}, 0.5))
	case CP_video:
		catprop(dircache.Category(FG_video, 0.5))
	case CP_audio:
		catprop(dircache.Category(FG_audio, 0.5))
	case CP_image:
		catprop(dircache.Category(FG_image, 0.5))
	case CP_books:
		catprop(dircache.Category(FG_books, 0.5))
	case CP_texts:
		catprop(dircache.Category(FG_texts, 0.5))
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrNotCat, EC_ctgrnotcat)
		return
	}

	userpath <- userfilepath{r, arg.PUID}
	Log.Printf("id%d: navigate to %s", acc.ID, catpath)
	WriteOK(w, ret)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid,omitempty"`
		Path string `json:"path,omitempty"`
	}
	var ret struct {
		List []Proper `json:"list"`
		PUID string   `json:"puid"`
		Path string   `json:"path"`
		Name string   `json:"shrname"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_folderbadreq)
			return
		}
		if len(arg.PUID) == 0 && len(arg.Path) == 0 {
			WriteError400(w, ErrArgNoPath, EC_foldernodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_foldernoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_foldernoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	var syspath string
	if len(arg.Path) > 0 {
		syspath = UnfoldPath(arg.Path)
		ret.PUID = pathcache.Cache(syspath)
	} else {
		var ok bool
		if syspath, ok = pathcache.Path(arg.PUID); !ok {
			WriteError(w, http.StatusNotFound, ErrNoPath, EC_foldernopath)
			return
		}
		ret.PUID = arg.PUID
	}

	var shrpath, base, cg = acc.GetSharePath(syspath, auth == acc)
	if cg.IsZero() {
		WriteError(w, http.StatusForbidden, ErrNoAccess, EC_folderaccess)
		return
	}
	ret.Path = shrpath
	ret.Name = PathBase(base)

	if ret.List, err = acc.Readdir(syspath, &cg); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_folderfail)
		return
	}
	userpath <- userfilepath{r, ret.PUID}
	Log.Printf("id%d: navigate to %s", acc.ID, syspath)

	WriteOK(w, ret)
}

// APIHANDLER
func ispathApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_ispathbadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_ispathnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_ispathnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_ispathdeny)
		return
	}

	prop, err := propcache.Get(arg.Path)
	WriteOK(w, prop)
}

// APIHANDLER
func shrlstApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret = []Proper{}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shrlstbadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_shrlstnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shrlstnoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	if auth != acc && !acc.IsShared(CP_shares) {
		WriteError(w, http.StatusForbidden, ErrNotShared, EC_shrlstnoshr)
		return
	}

	ret = acc.ScanShares()
	WriteOK(w, ret)
}

// APIHANDLER
func shraddApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shraddbadreq)
			return
		}
		if len(arg.PUID) == 0 {
			WriteError400(w, ErrArgNoPath, EC_shraddnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_shraddnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shraddnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_shradddeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_shraddnopath)
	}
	if !acc.PathAdmin(syspath) {
		WriteError(w, http.StatusForbidden, ErrNoAccess, EC_shraddaccess)
		return
	}

	var ret = acc.AddShare(syspath)
	Log.Printf("id%d: add share '%s' as %s", acc.ID, syspath, arg.PUID)

	WriteOK(w, ret)
}

// APIHANDLER
func shrdelApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}
	var ok bool

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shrdelbadreq)
			return
		}
		if len(arg.PUID) == 0 {
			WriteError400(w, ErrArgNoPath, EC_shrdelnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_shrdelnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shrdelnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_shrdeldeny)
		return
	}

	if ok = acc.DelShare(arg.PUID); ok {
		Log.Printf("id%d: delete share %s", acc.ID, arg.PUID)
	}

	WriteOK(w, ok)
}

// APIHANDLER
func drvlstApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}
	var ret []Proper

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_drvlstbadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_drvlstnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvlstnoacc)
		return
	}
	var auth *Account
	if auth, err = GetAuth(r); err != nil {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	if auth != acc && !acc.IsShared(CP_drives) {
		WriteError(w, http.StatusForbidden, ErrNotShared, EC_drvlstnoshr)
		return
	}

	ret = acc.ScanRoots()
	Log.Printf("id%d: navigate to drives", acc.ID)
	WriteOK(w, ret)
}

// APIHANDLER
func drvaddApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_drvaddbadreq)
			return
		}
		if len(arg.Path) == 0 {
			WriteError400(w, ErrArgNoPath, EC_drvaddnodata)
			return
		}
		arg.Path = filepath.ToSlash(arg.Path)
		if arg.Path[len(arg.Path)-1] != '/' {
			arg.Path += "/"
		}
	} else {
		WriteError400(w, ErrNoJson, EC_drvaddnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvaddnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_drvadddeny)
		return
	}

	if acc.RootIndex(arg.Path) >= 0 {
		WriteOK(w, nil)
		return
	}

	var dk DriveKit
	dk.Setup(arg.Path)
	if err = dk.Scan(arg.Path); err == nil {
		WriteError400(w, err, EC_drvaddfile)
		return
	}

	acc.mux.Lock()
	acc.Roots = append(acc.Roots, arg.Path)
	acc.mux.Unlock()

	WriteOK(w, dk)
}

// APIHANDLER
func drvdelApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_drvdelbadreq)
			return
		}
		if len(arg.PUID) == 0 {
			WriteError400(w, ErrArgNoPath, EC_drvdelnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_drvdelnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(arg.AID); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvdelnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_drvdeldeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_drvdelnopath)
	}

	var i int
	if i = acc.RootIndex(syspath); i >= 0 {
		acc.mux.Lock()
		acc.Roots = append(acc.Roots[:i], acc.Roots[i+1:]...)
		acc.mux.Unlock()
	}

	WriteOK(w, i >= 0)
}

// The End.
