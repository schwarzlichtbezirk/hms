package hms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HTTP error messages
const (
	ErrDeny    = "access denied"
	ErrShrNone = "404 share not found"
	ErrShrGone = "410 share is closed and does not available any more"
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
			http.NotFound(w, r)
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
		http.NotFound(w, r)
	}
}

// APIHANDLER
func shareHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&sharecallcount, 1)

	var path = r.URL.Path[len(shareprefix):]
	var prefend = strings.IndexByte(path, '/')
	var pref, suff string
	if prefend == -1 {
		pref, suff = path, ""
	} else {
		pref, suff = path[:prefend], path[prefend+1:]
	}

	shrmux.RLock()
	var shr, ok = sharespref[pref]
	shrmux.RUnlock()
	if !ok {
		shrmux.RLock()
		_, ok = sharesgone[pref]
		shrmux.RUnlock()
		if ok {
			WriteJson(w, http.StatusGone, &AjaxErr{ErrShrGone, EC_sharegone})
			return
		} else {
			WriteJson(w, http.StatusNotFound, &AjaxErr{ErrShrNone, EC_sharenone})
			return
		}
	}

	WriteStdHeader(w)
	http.ServeFile(w, r, shr.Path+suff)
}

// APIHANDLER
func localHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&localcallcount, 1)

	if !IsAdmin(r) {
		WriteJson(w, http.StatusUnauthorized, &AjaxErr{ErrDeny, EC_localunauth})
		return
	}
	var path = r.FormValue("path")
	if len(path) == 0 {
		WriteJson(w, http.StatusNotAcceptable, &AjaxErr{"'path' argument required", EC_localnopath})
		return
	}

	WriteStdHeader(w)
	http.ServeFile(w, r, path)
}

// APIHANDLER
func pingApi(w http.ResponseWriter, r *http.Request) {
	var body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError500(w, err, EC_pingbadreq)
		return
	}

	WriteJson(w, http.StatusOK, body)
}

// APIHANDLER
func reloadApi(w http.ResponseWriter, r *http.Request) {
	var err error
	type cached struct {
		Prefix string  `json:"prefix"`
		Count  uint64  `json:"count"`
		Size   uint64  `json:"size"`
		Errors []error `json:"errors"`
	}

	var body []byte
	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError500(w, err, EC_reloadbadreq)
		return
	}

	var prefixlist []string
	err = json.Unmarshal(body, &prefixlist)
	if err != nil {
		WriteError400(w, err.Error(), EC_reloadbadcontent)
		return
	}

	var p []cached
	for _, prefix := range prefixlist {
		var path, ok = routedpaths[prefix]
		if !ok {
			path, ok = routedpaths["/"+prefix+"/"]
			if !ok {
				WriteError400(w, fmt.Sprintf("given routes prefix \"%s\" does not assigned to any file path", prefix), EC_reloadbadprefix)
				return
			}
			prefix = "/" + prefix + "/"
		}
		var res cached
		res.Prefix = prefix
		res.Count, res.Size, res.Errors = LoadFiles(path, prefix)
		p = append(p, res)
		LogErrors(res.Errors)
		Log.Printf("reloaded cache of %d files on %d bytes for %s route", res.Count, res.Size, prefix)
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func servinfoApi(w http.ResponseWriter, r *http.Request) {
	var p = map[string]interface{}{
		"started":  starttime.UnixNano() / int64(time.Millisecond),
		"govers":   runtime.Version(),
		"os":       runtime.GOOS,
		"numcpu":   runtime.NumCPU(),
		"maxprocs": runtime.GOMAXPROCS(0),
		"destpath": destpath,
		"rootpath": rootpath,
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func memusageApi(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var p = map[string]interface{}{
		"running":       time.Since(starttime) / time.Millisecond,
		"heapalloc":     mem.HeapAlloc,
		"heapsys":       mem.HeapSys,
		"totalalloc":    mem.TotalAlloc,
		"nextgc":        mem.NextGC,
		"numgc":         mem.NumGC,
		"pausetotalns":  mem.PauseTotalNs,
		"gccpufraction": mem.GCCPUFraction,
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func getlogApi(w http.ResponseWriter, r *http.Request) {
	var err error

	var size = Log.Size()

	// Get number of log lines
	var num int
	if s := r.FormValue("num"); len(s) > 0 {
		var i64 int64
		if i64, err = strconv.ParseInt(s, 10, 64); err != nil {
			WriteError400(w, "'num' parameter not recognized", EC_getlogbadnum)
			return
		}
		num = int(i64)
	}
	if num <= 0 || num > size {
		num = size
	}

	var p = make([]interface{}, num)
	var h = Log.Ring()
	for i := 0; i < num; i++ {
		p[i] = h.Value
		h = h.Prev()
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func getdrvApi(w http.ResponseWriter, r *http.Request) {
	if !IsAdmin(r) {
		WriteJson(w, http.StatusUnauthorized, &AjaxErr{ErrDeny, EC_getdrvunauth})
		return
	}

	var p = getdrives()
	Log.Printf("navigate to root")

	if r.Method == "HEAD" {
		WriteStdHeader(w)
		return
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var p []IFileProp
	var adm = IsAdmin(r)

	var path = r.FormValue("path")
	var sval = r.FormValue("sort")

	if len(path) == 0 {
		// does not give anything here if it is not admin
		if adm {
			p = getdrives()
		} else {
			p = []IFileProp{}
		}
	} else {
		p, err = readdir(path)
		if err != nil {
			WriteJson(w, http.StatusNotFound, &AjaxErr{err.Error(), EC_folderabsent})
			return
		}
		switch sval {
		case "name":
			sort.Slice(p, func(i, j int) bool {
				var pi = p[i].Base()
				var pj = p[j].Base()
				if (pi.Type == Dir) != (pj.Type == Dir) {
					return pi.Type == Dir
				} else {
					return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
				}
			})
		case "type":
			sort.Slice(p, func(i, j int) bool {
				var pi = p[i].Base()
				var pj = p[j].Base()
				if pi.Type != pj.Type {
					return pi.Type < pj.Type
				} else {
					return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
				}
			})
		case "size":
			sort.Slice(p, func(i, j int) bool {
				var pi = p[i].Base()
				var pj = p[j].Base()
				if (pi.Type == Dir) != (pj.Type == Dir) {
					return pi.Type < pj.Type
				} else {
					if pi.Type == Dir {
						return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
					} else {
						return pi.Size < pj.Size
					}
				}
			})
		}
	}
	Log.Printf("navigate to: %s", path)

	if r.Method == "HEAD" {
		WriteStdHeader(w)
		return
	}

	WriteJson(w, http.StatusOK, p)
}

// APIHANDLER
func sharedApi(w http.ResponseWriter, r *http.Request) {
	shrmux.RLock()
	var b, _ = json.Marshal(shareslist)
	shrmux.RUnlock()

	WriteJson(w, http.StatusOK, b)
}

// APIHANDLER
func addshrApi(w http.ResponseWriter, r *http.Request) {
	if !IsAdmin(r) {
		WriteJson(w, http.StatusUnauthorized, &AjaxErr{ErrDeny, EC_addshrunauth})
		return
	}

	var fpath = r.FormValue("path")
	if len(fpath) == 0 {
		WriteJson(w, http.StatusNotAcceptable, &AjaxErr{"'path' argument required", EC_addshrnopath})
		return
	}

	shrmux.RLock()
	var _, shrok = sharespath[fpath]
	shrmux.RUnlock()
	if shrok { // share already added
		WriteJson(w, http.StatusOK, []byte("null"))
		return
	}

	var f, err = os.Open(fpath)
	if err != nil {
		WriteJson(w, http.StatusNotFound, &AjaxErr{err.Error(), EC_addshrbadpath})
		return
	}
	var fi, _ = f.Stat()
	f.Close()

	var shr FileProp
	shr.Setup(fi)
	shr.Path = fpath
	shr.MakeShare()

	WriteJson(w, http.StatusOK, shr)
}

// APIHANDLER
func delshrApi(w http.ResponseWriter, r *http.Request) {
	if !IsAdmin(r) {
		WriteJson(w, http.StatusUnauthorized, &AjaxErr{ErrDeny, EC_delshrunauth})
		return
	}

	var ok bool
	if pref := r.FormValue("pref"); len(pref) > 0 {
		ok = DelSharePref(pref)
	} else if path := r.FormValue("path"); len(path) > 0 {
		ok = DelSharePath(path)
	} else {
		WriteJson(w, http.StatusNotAcceptable, &AjaxErr{"'pref' or 'path' argument required", EC_delshrnopath})
		return
	}

	WriteJson(w, http.StatusOK, ok)
}

// The End.
