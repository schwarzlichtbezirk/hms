package hms

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
)

//////////////////////////
// API request handlers //
//////////////////////////

// APIHANDLER
func pageHandler(pref, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteHtmlHeader(w)
		var content, ok = filecache[pref+routedpages[name]]
		if ok {
			pccmux.Lock()
			pagecallcount[name]++
			pccmux.Unlock()

			http.ServeContent(w, r, routedpages[name], starttime, bytes.NewReader(content))
		} else {
			WriteJson(w, http.StatusNotFound, &AjaxErr{ErrNotFound, EC_pageabsent})
		}
	}
}

// APIHANDLER
func filecacheHandler(w http.ResponseWriter, r *http.Request) {
	var route = r.URL.Path
	var content, ok = filecache[route]
	WriteStdHeader(w)
	if ok {
		if strings.HasPrefix(route, "/plug/") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		if ct, ok := mimeext[strings.ToLower(filepath.Ext(route))]; ok {
			w.Header().Set("Content-Type", ct)
		}
		http.ServeContent(w, r, route, starttime, bytes.NewReader(content))
	} else {
		WriteJson(w, http.StatusNotFound, &AjaxErr{ErrNotFound, EC_fileabsent})
	}
}

// APIHANDLER
func fileHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&sharecallcount, 1)
	var err error

	var shr string
	if shr, err = url.QueryUnescape(r.URL.Path[len(shareprefix):]); err != nil {
		WriteError400(w, err, EC_filebadurl)
		return
	}
	var path, shared = checksharepath(shr)
	if !shared {
		if err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		return
	}

	if _, ok := r.Header["If-Range"]; !ok { // not partial content
		Log.Printf("serve: %s", filepath.Base(path))
	}

	if ct, ok := mimeext[strings.ToLower(filepath.Ext(path))]; ok {
		w.Header().Set("Content-Type", ct)
	}
	WriteStdHeader(w)
	http.ServeFile(w, r, path)
}

// APIHANDLER
func pingApi(w http.ResponseWriter, r *http.Request) {
	var body, _ = ioutil.ReadAll(r.Body)
	WriteJson(w, http.StatusOK, body)
}

// APIHANDLER
func reloadApi(w http.ResponseWriter, r *http.Request) {
	type cached struct {
		Prefix string  `json:"prefix"`
		Count  uint64  `json:"count"`
		Size   uint64  `json:"size"`
		Errors []error `json:"errors"`
	}

	var err error
	var arg []string
	var ret []cached

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_reloadbadreq)
			return
		}
		if len(arg) == 0 {
			WriteError400(w, ErrNoData, EC_reloadnodata)
			return
		}
	} else {
		WriteError500(w, ErrNoJson, EC_reloadnoreq)
		return
	}

	var reloadtpl = false
	for _, prefix := range arg {
		var path, ok = routedpaths[prefix]
		if !ok {
			prefix = "/" + prefix + "/"
			path, ok = routedpaths[prefix]
			if !ok {
				WriteError400(w, fmt.Errorf("given routes prefix \"%s\" does not assigned to any file path", prefix), EC_reloadbadprf)
				return
			}
		}
		var res cached
		res.Prefix = prefix
		res.Count, res.Size, res.Errors = LoadFiles(path, prefix)
		ret = append(ret, res)
		LogErrors(res.Errors)
		Log.Printf("reloaded cache of %d files on %d bytes for %s route", res.Count, res.Size, prefix)
		if prefix == "/devm/" || prefix == "/relm/" {
			reloadtpl = true
		}
	}
	if reloadtpl {
		loadtemplates()
	}

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func servinfoApi(w http.ResponseWriter, r *http.Request) {
	var ret = map[string]interface{}{
		"started":  UnixJS(starttime),
		"govers":   runtime.Version(),
		"os":       runtime.GOOS,
		"numcpu":   runtime.NumCPU(),
		"maxprocs": runtime.GOMAXPROCS(0),
		"destpath": destpath,
		"rootpath": rootpath,
	}

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func memusageApi(w http.ResponseWriter, r *http.Request) {
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
func getdrvApi(w http.ResponseWriter, r *http.Request) {
	var ret = scanroots()
	Log.Printf("navigate to root")
	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var ret folderRet

	// get arguments
	var argpath = filepath.ToSlash(r.FormValue("path"))

	var isroot = len(argpath) == 0
	var fpath, shared = checksharepath(argpath)

	var admerr = CheckAuth(r)
	if admerr != nil && !isroot && !shared {
		WriteJson(w, http.StatusUnauthorized, admerr)
		return
	}

	if isroot {
		if admerr == nil {
			ret.Paths = scanroots()
		}

		if admerr == nil || ShowSharesUser {
			shrmux.RLock()
			for _, fpath := range sharespref {
				if cp, err := propcache.Get(fpath); err == nil {
					ret.AddProp(cp.(FileProper))
				}
			}
			shrmux.RUnlock()
		}
	} else {
		if ret, err = readdir(fpath); err != nil {
			WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_folderfail})
			return
		}
	}
	if isroot {
		Log.Printf("navigate to root")
	} else {
		Log.Printf("navigate to: %s", fpath)
	}

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func purgeApi(w http.ResponseWriter, r *http.Request) {
	propcache.Purge()
	thumbcache.Purge()
	registershares()

	WriteJson(w, http.StatusOK, nil)
}

// APIHANDLER
func shrlstApi(w http.ResponseWriter, r *http.Request) {
	shrmux.RLock()
	var lst = make([]FileProper, len(sharespref))
	var i int
	for _, fpath := range sharespref {
		if cp, err := propcache.Get(fpath); err == nil {
			lst[i] = cp.(FileProper)
		}
		i++
	}
	shrmux.RUnlock()
	var jb, _ = json.Marshal(lst)

	WriteJson(w, http.StatusOK, jb)
}

// APIHANDLER
func shraddApi(w http.ResponseWriter, r *http.Request) {
	// get arguments
	var path string
	if path = filepath.ToSlash(r.FormValue("path")); len(path) == 0 {
		WriteError400(w, ErrArgNoPath, EC_addshrnopath)
		return
	}

	shrmux.RLock()
	var _, has = sharespath[path]
	shrmux.RUnlock()
	if has { // share already added
		WriteJson(w, http.StatusOK, []byte("null"))
		return
	}

	var fi, err = FileStat(path)
	if err != nil {
		WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_addshrbadpath})
		return
	}

	var prop = MakeProp(path, fi)
	MakeShare(path, prop)

	WriteJson(w, http.StatusOK, prop)
}

// APIHANDLER
func shrdelApi(w http.ResponseWriter, r *http.Request) {
	var ok bool

	// get arguments & process
	var pref, path string
	if pref = r.FormValue("pref"); len(pref) > 0 {
		ok = DelSharePref(pref)
	} else if path = filepath.ToSlash(r.FormValue("path")); len(path) > 0 {
		ok = DelSharePath(path)
	} else {
		WriteError400(w, ErrArgNoPref, EC_delshrnopath)
		return
	}

	WriteJson(w, http.StatusOK, ok)
}

// The End.
