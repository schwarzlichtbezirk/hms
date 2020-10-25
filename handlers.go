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

	var spath string
	if hpath, ok := pathcache.Path(chunks[3]); ok {
		if len(chunks) > 3 {
			spath = filepath.ToSlash(filepath.Join(hpath, strings.Join(chunks[4:], "/")))
		} else {
			spath = hpath
		}
	} else {
		spath = strings.Join(chunks[3:], "/")
	}

	var syspath = acc.GetSystemPath(spath)
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
	if syspath, ok := pathcache.Path(chunks[3]); ok {
		var state = acc.PathState(syspath)
		if state == FPA_none {
			WriteError(w, http.StatusForbidden, ErrFpaNone, EC_thumbndeny)
			return
		}
	} else {
		WriteError(w, http.StatusNotFound, ErrNoPath, EC_thumbnopath)
		return
	}

	var val interface{}
	if val, err = thumbcache.Get(puid); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_thumbabsent)
		return
	}
	var tmb, ok = val.(*ThumbElem)
	if !ok {
		WriteError500(w, ErrBadThumb, EC_thumbbadcnt)
		return
	}
	if tmb == nil {
		WriteError(w, http.StatusNotFound, ErrNotThumb, EC_thumbnotcnt)
		return
	}
	w.Header().Set("Content-Type", tmb.Mime)
	http.ServeContent(w, r, puid, starttime, bytes.NewReader(tmb.Data))
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
func reloadApi(w http.ResponseWriter, r *http.Request) {
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
		var tmb = v.(*ThumbElem)
		var s *stat
		switch tmb.Mime {
		case "image/gif":
			s = &gif
		case "image/png":
			s = &png
		case "image/jpeg":
			s = &jpg
		default:
			panic("unexpected MIME type in cache " + tmb.Mime)
		}
		var l = float64(len(tmb.Data))
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
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var arg struct {
		AID  int    `json:"aid"`
		PUID string `json:"puid"`
	}
	var ret struct {
		List  []ShareKit `json:"list"`
		Path  string     `json:"path"`
		State int        `json:"state"`
		Name  string     `json:"shrname"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_folderbadreq)
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

	if len(arg.PUID) == 0 { // for root only
		var auth, _ = CheckAuth(r)
		if acc == auth {
			ret.List = auth.ScanRoots()
			ret.State = FPA_admin
		} else {
			ret.List = []ShareKit{}
			ret.State = FPA_share
		}

		if acc == auth || cfg.ShowSharesUser {
			acc.mux.RLock()
			for _, path := range acc.Shares {
				if prop, err := propcache.Get(path); err == nil {
					var sk = ShareKit{prop.(Proper), "", ""}
					ret.List = append(ret.List, sk)
				}
			}
			acc.mux.RUnlock()
		}
		Log.Printf("navigate to root")

		WriteOK(w, ret)
		return
	}

	var syspath, ok = pathcache.Path(arg.PUID)
	if !ok {
		WriteError400(w, ErrNoPath, EC_foldernopath)
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
		var share, _ = acc.GetSharePath(syspath)
		ret.Path = share
		ret.Name = filepath.Base(share)
	} else {
		ret.Path = syspath
	}
	ret.State = state

	if ret.List, err = acc.Readdir(syspath); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_folderfail)
		return
	}
	Log.Printf("navigate to: %s", syspath)

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
	var ret []Proper

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
	if auth, err = CheckAuth(r); !(auth == acc || cfg.ShowSharesUser) {
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
		Path string `json:"path"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shraddbadreq)
			return
		}
		if len(arg.Path) == 0 {
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

	var syspath = acc.GetSystemPath(arg.Path)
	var state = acc.PathState(syspath)
	if state == FPA_none {
		WriteError(w, http.StatusForbidden, ErrFpaNone, EC_filefpanone)
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
		Path string `json:"path,omitempty"`
		PUID string `json:"puid,omitempty"`
	}
	var ok bool

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shrdelbadreq)
			return
		}
		if len(arg.Path) == 0 && len(arg.PUID) == 0 {
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

	if len(arg.PUID) > 0 {
		if ok = acc.DelShareHash(arg.PUID); ok {
			Log.Printf("id%d: delete share %s", acc.ID, arg.PUID)
		}
	} else if len(arg.Path) > 0 {
		if ok = acc.DelSharePath(arg.Path); ok {
			Log.Printf("id%d: delete share %s", acc.ID, arg.Path)
		}
	} else {
		WriteError400(w, ErrArgNoHash, EC_shrdelnohash)
		return
	}

	WriteOK(w, ok)
}

// APIHANDLER
func drvlstApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var ret = auth.ScanRoots()
	Log.Printf("navigate to root")
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
		Path string `json:"path"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_drvdelbadreq)
			return
		}
		if len(arg.Path) == 0 {
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

	var i int
	if i = acc.RootIndex(arg.Path); i >= 0 {
		acc.mux.Lock()
		acc.Roots = append(acc.Roots[:i], acc.Roots[i+1:]...)
		acc.mux.Unlock()
	}

	WriteOK(w, i >= 0)
}

// The End.
