package hms

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
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
	ErrArgNoPref = errors.New("'pref' or 'path' argument required")
	ErrNotPath   = errors.New("path is not directory")
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
			WriteJson(w, http.StatusNotFound, &AjaxErr{ErrNotFound, EC_pageabsent})
		}
	}
}

// APIHANDLER
func fileHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&sharecallcount, 1)

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}

	var aid uint64
	var err error
	if aid, err = strconv.ParseUint(chunks[1][2:], 10, 32); err != nil {
		WriteError400(w, err, EC_filebadaccid)
		return
	}

	var acc *Account
	if acc = AccList.ByID(int(aid)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_filenoacc)
		return
	}

	var syspath, shared = acc.CheckSharePath(strings.Join(chunks[3:], "/"))
	if !shared {
		var auth *Account
		if auth, err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		if acc.ID != auth.ID {
			WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_filedeny})
			return
		}
	}

	if _, ok := r.Header["If-Range"]; !ok { // not partial content
		Log.Printf("id%d: serve %s", acc.ID, filepath.Base(syspath))
	}

	WriteStdHeader(w)
	http.ServeFile(w, r, syspath)
}

// APIHANDLER
func pingApi(w http.ResponseWriter, r *http.Request) {
	var body, _ = ioutil.ReadAll(r.Body)
	WriteJson(w, http.StatusOK, body)
}

// APIHANDLER
func purgeApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	propcache.Purge()
	thumbcache.Purge()

	AccList.mux.RLock()
	for _, acc := range AccList.list {
		acc.UpdateShares()
	}
	AccList.mux.RUnlock()

	WriteJson(w, http.StatusOK, nil)
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

	WriteJson(w, http.StatusOK, &datapack.PackHdr)
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

	WriteJson(w, http.StatusOK, ret)
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

	WriteJson(w, http.StatusOK, ret)
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

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
	}
	var ret []ShareKit

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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_foldernoacc)
		return
	}

	var isroot = len(arg.Path) == 0
	var _, shared = acc.CheckSharePath(arg.Path)

	var auth *Account
	if auth, err = CheckAuth(r); err != nil && !isroot && !shared {
		WriteJson(w, http.StatusUnauthorized, err)
		return
	}

	if isroot {
		if auth != nil {
			ret = auth.ScanRoots()
		}

		if acc == auth || ShowSharesUser {
			acc.mux.RLock()
			for pref, syspath := range acc.sharespref {
				if cp, err := propcache.Get(syspath); err == nil {
					var sk = ShareKit{cp.(Proper), "", ""}
					sk.SetPref(pref)
					ret = append(ret, sk)
				}
			}
			acc.mux.RUnlock()
		}
		Log.Printf("navigate to root")
	} else {
		if ret, err = acc.Readdir(arg.Path); err != nil {
			WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_folderfail})
			return
		}
		Log.Printf("navigate to: %s", arg.Path)
	}

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func ispathApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path"`
		Prop bool   `json:"prop,omitempty"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_ispathbadreq)
			return
		}
		if len(arg.Path) == 0 {
			WriteError400(w, ErrArgNoPath, EC_ispathnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_ispathnoreq)
		return
	}

	var acc *Account
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_ispathnoacc)
		return
	}
	if auth != acc {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_ispathdeny})
		return
	}

	var fi os.FileInfo
	var ret interface{}
	if fi, err = os.Stat(arg.Path); err == nil {
		if arg.Prop {
			var sk = ShareKit{MakeProp(arg.Path, fi), arg.Path, ""}
			acc.SetupPref(&sk, arg.Path)
			ret = sk
		} else {
			ret = true
		}
	} else {
		if os.IsNotExist(err) {
			ret = false
		} else {
			ret = nil
		}
	}
	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func shrlstApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID int `json:"aid"`
	}

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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shrlstnoacc)
		return
	}

	var auth *Account
	if auth, err = CheckAuth(r); !(auth == acc || ShowSharesUser) {
		if err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		} else {
			WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_shrlstdeny})
			return
		}
	}

	acc.mux.RLock()
	var lst = make([]ShareKit, len(acc.sharespref))
	var i int
	for pref, fpath := range acc.sharespref {
		var prop Proper
		var cp interface{}
		if cp, err = propcache.Get(fpath); err == nil {
			prop = cp.(Proper)
		} else {
			var fi os.FileInfo
			if fi, err = os.Stat(fpath); err != nil {
				continue
			}
			prop = MakeProp(fpath, fi)
		}
		var sk = ShareKit{prop, fpath, ""}
		sk.SetPref(pref)
		lst[i] = sk
		i++
	}
	acc.mux.RUnlock()

	WriteJson(w, http.StatusOK, lst)
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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shraddnoacc)
		return
	}
	if auth != acc {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_shradddeny})
		return
	}

	acc.mux.RLock()
	var _, has = acc.sharespath[arg.Path]
	acc.mux.RUnlock()
	if has { // share already added
		WriteJson(w, http.StatusOK, []byte("null"))
		return
	}

	var fpath = acc.GetSharePath(arg.Path)

	var fi os.FileInfo
	if fi, err = os.Stat(fpath); err != nil {
		if os.IsNotExist(err) {
			WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_shraddnopath})
			return
		} else {
			WriteError500(w, err, EC_shraddbadpath)
			return
		}
	}

	var pref = acc.MakeShare(fi.Name(), fpath)

	var sk = ShareKit{MakeProp(fpath, fi), arg.Path, ""}
	sk.SetPref(pref)

	Log.Printf("id%d: add share %s as %s", acc.ID, fpath, pref)

	WriteJson(w, http.StatusOK, sk)
}

// APIHANDLER
func shrdelApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var err error
	var arg struct {
		AID  int    `json:"aid"`
		Path string `json:"path,omitempty"`
		Pref string `json:"pref,omitempty"`
	}
	var ok bool

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_shrdelbadreq)
			return
		}
		if len(arg.Path) == 0 && len(arg.Pref) == 0 {
			WriteError400(w, ErrArgNoPath, EC_shrdelnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_shrdelnoreq)
		return
	}

	var acc *Account
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_shrdelnoacc)
		return
	}
	if auth != acc {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_shrdeldeny})
		return
	}

	if len(arg.Pref) > 0 {
		if ok = acc.DelSharePref(arg.Pref); ok {
			Log.Printf("id%d: delete share %s", acc.ID, arg.Pref)
		}
	} else if len(arg.Path) > 0 {
		if ok = acc.DelSharePath(arg.Path); ok {
			Log.Printf("id%d: delete share %s", acc.ID, arg.Path)
		}
	} else {
		WriteError400(w, ErrArgNoPref, EC_shrdelnopath)
		return
	}

	WriteJson(w, http.StatusOK, ok)
}

// APIHANDLER
func drvlstApi(w http.ResponseWriter, r *http.Request, auth *Account) {
	var ret = auth.ScanRoots()
	Log.Printf("navigate to root")
	WriteJson(w, http.StatusOK, ret)
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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvaddnoacc)
		return
	}
	if auth != acc {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_drvadddeny})
		return
	}

	if acc.RootIndex(arg.Path) >= 0 {
		WriteJson(w, http.StatusOK, nil)
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

	var sk = ShareKit{&dk, arg.Path, ""}

	WriteJson(w, http.StatusOK, sk)
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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_drvdelnoacc)
		return
	}
	if auth != acc {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_drvdeldeny})
		return
	}

	var i int
	if i = acc.RootIndex(arg.Path); i >= 0 {
		acc.mux.Lock()
		acc.Roots = append(acc.Roots[:i], acc.Roots[i+1:]...)
		acc.mux.Unlock()
	}

	WriteJson(w, http.StatusOK, i >= 0)
}

// The End.
