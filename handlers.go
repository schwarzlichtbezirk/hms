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
	ErrFpaNone   = errors.New("account has no access to specified file path")
	ErrFpaAdmin  = errors.New("not authorized for access to specified file path")
)

//////////////////////////
// API request handlers //
//////////////////////////

// APIHANDLER
func pageHandler(pref, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var alias = pagealias[name]
		if content, ok := pagecache[pref+alias]; ok {
			pccmux.Lock()
			pagecallcount[name]++
			pccmux.Unlock()

			WriteHtmlHeader(w)
			http.ServeContent(w, r, alias, starttime, bytes.NewReader(content))
		} else {
			WriteError(w, http.StatusNotFound, ErrNotFound, EC_pageabsent)
		}
	}
}

// APIHANDLER
func fileHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&sharecallcount, 1)

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

	var syspath = UnfoldPath(strings.Join(chunks[3:], "/"))
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_filefpanone)
		return
	}
	if state == FPA_admin {
		var auth *Account
		if auth, err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		if acc.ID != auth.ID {
			WriteError(w, http.StatusForbidden, ErrFpaAdmin, EC_filefpaadmin)
			return
		}
	}

	if _, ok := r.Header["If-Range"]; !ok { // not partial content
		Log.Printf("id%d: serve %s", acc.ID, filepath.Base(syspath))
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

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_medianopath)
		return
	}
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_mediafpanone)
		return
	}
	if state == FPA_admin {
		var auth *Account
		if auth, err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		if acc.ID != auth.ID {
			WriteError(w, http.StatusForbidden, ErrFpaAdmin, EC_mediafpaadmin)
			return
		}
	}

	var val interface{}
	if val, err = mediacache.Get(puid); err != nil {
		if !errors.Is(err, ErrUncacheable) {
			WriteError(w, http.StatusNotFound, err, EC_mediaabsent)
			return
		}

		if _, ok := r.Header["If-Range"]; !ok { // not partial content
			Log.Printf("id%d: serve %s", acc.ID, filepath.Base(syspath))
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
		Log.Printf("id%d: media %s", acc.ID, filepath.Base(syspath))
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

	var puid = chunks[3]
	var syspath, ok = pathcache.Path(puid)
	if !ok {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_thumbnopath)
		return
	}
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_thumbfpanone)
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_homenoacc)
		return
	}
	var auth, _ = CheckAuth(r)

	if auth == acc {
		ret = append(ret, NewCatKit("Drives list", "drives"))
	}
	if auth == acc || acc.ShowShares {
		ret = append(ret, NewCatKit("Shared resources", "shares"))
	}

	Log.Printf("id%d: navigate to home", acc.ID)
	WriteOK(w, ret)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid,omitempty"`
		Path string `json:"path,omitempty"`
	}
	var ret struct {
		List  []Proper `json:"list"`
		PUID  string   `json:"puid"`
		Path  string   `json:"path"`
		State int      `json:"state"`
		Name  string   `json:"shrname"`
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_foldernoacc)
		return
	}

	var syspath string
	if len(arg.PUID) > 0 {
		var ok bool
		if syspath, ok = pathcache.Path(arg.PUID); !ok {
			WriteError400(w, ErrNoPath, EC_foldernopath)
			return
		}
		ret.PUID = arg.PUID
	} else {
		syspath = UnfoldPath(arg.Path)
		ret.PUID = pathcache.Cache(syspath)
	}
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_folderfpanone)
		return
	}
	if state == FPA_admin {
		var auth *Account
		if auth, err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		if acc.ID != auth.ID {
			WriteError(w, http.StatusForbidden, ErrFpaAdmin, EC_folderfpaadmin)
			return
		}
	}
	if state == FPA_share {
		var path, share = acc.GetSharePath(syspath)
		ret.Path = path
		ret.Name = filepath.Base(share)
	} else {
		ret.Path = syspath
	}
	ret.State = state

	if ret.List, err = acc.Readdir(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_folderfail)
		return
	}
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shrlstnoacc)
		return
	}

	var auth *Account
	if auth, err = CheckAuth(r); !(auth == acc || acc.ShowShares) {
		if err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		} else {
			WriteError(w, http.StatusForbidden, ErrDeny, EC_shrlstdeny)
			return
		}
	}

	acc.mux.RLock()
	for _, path := range acc.Shares {
		if prop, err := propcache.Get(path); err == nil {
			ret = append(ret, prop.(Proper))
		}
	}
	acc.mux.RUnlock()

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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shraddnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_shradddeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError400(w, ErrNoPath, EC_shraddnopath)
	}
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_shraddfpanone)
		return
	}

	var ret = acc.AddShare(syspath)
	Log.Printf("id%d: add share %s", acc.ID, syspath)

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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
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
func drvlstApi(w http.ResponseWriter, r *http.Request, auth *Account) {
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvlstnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_drvlstdeny)
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
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
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvdelnoacc)
		return
	}
	if auth != acc {
		WriteError(w, http.StatusForbidden, ErrDeny, EC_drvdeldeny)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError400(w, ErrNoPath, EC_drvdelnopath)
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
