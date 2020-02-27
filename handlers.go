package hms

import (
	"bytes"
	"encoding/json"
	"errors"
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
var (
	ErrNoJson = errors.New("data not given")
	ErrNoData = errors.New("data is empty")

	ErrDeny      = errors.New("access denied")
	ErrShareNone = errors.New("404 share not found")
	ErrShareGone = errors.New("410 share is closed and does not available any more")
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
			WriteJson(w, http.StatusGone, &AjaxErr{ErrShareGone, EC_sharegone})
			return
		} else {
			WriteJson(w, http.StatusNotFound, &AjaxErr{ErrShareNone, EC_sharenone})
			return
		}
	}

	WriteStdHeader(w)
	http.ServeFile(w, r, shr.Path+suff)
}

// APIHANDLER
func localHandler(w http.ResponseWriter, r *http.Request) {
	incuint(&localcallcount, 1)

	// get arguments
	var path = r.FormValue("path")
	if len(path) == 0 {
		WriteError400(w, ErrArgNoPath, EC_localnopath)
		return
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
	var ret = getdrives()
	Log.Printf("navigate to root")
	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func folderApi(w http.ResponseWriter, r *http.Request) {
	incuint(&foldercallcout, 1)

	var err error
	var ret folderRet

	// get arguments
	var path = r.FormValue("path")
	var sval = r.FormValue("sort")

	if len(path) == 0 {
		ret.Paths = getdrives()

		shrmux.RLock()
		var lst = make([]*FileProp, len(shareslist))
		copy(lst, shareslist)
		shrmux.RUnlock()

		for _, fp := range lst {
			if fp.Type == Dir {
				ret.Paths = append(ret.Paths, DirProp{
					FileProp: *fp,
				})
			} else {
				ret.Files = append(ret.Files, *fp)
			}
		}
	} else {
		ret, err = readdir(path)
		if err != nil {
			WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_folderabsent})
			return
		}
		switch sval {
		case "name":
			sort.Slice(ret.Files, func(i, j int) bool {
				var pi = ret.Files[i]
				var pj = ret.Files[j]
				if (pi.Type == Dir) != (pj.Type == Dir) {
					return pi.Type == Dir
				} else {
					return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
				}
			})
		case "type":
			sort.Slice(ret.Files, func(i, j int) bool {
				var pi = ret.Files[i]
				var pj = ret.Files[j]
				if pi.Type != pj.Type {
					return pi.Type < pj.Type
				} else {
					return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
				}
			})
		case "size":
			sort.Slice(ret.Files, func(i, j int) bool {
				var pi = ret.Files[i]
				var pj = ret.Files[j]
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
		// arrange folders by name on any case
		if len(sval) > 0 {
			sort.Slice(ret.Paths, func(i, j int) bool {
				var pi = ret.Paths[i]
				var pj = ret.Paths[j]
				return strings.ToLower(pi.Name) < strings.ToLower(pj.Name)
			})
		}
		// cache folder thumnails
		go func() {
			for _, fp := range ret.Files {
				CacheImg(&fp)
			}
		}()
	}
	Log.Printf("navigate to: %s", path)

	WriteJson(w, http.StatusOK, ret)
}

// APIHANDLER
func shrlstApi(w http.ResponseWriter, r *http.Request) {
	shrmux.RLock()
	var jb, _ = json.Marshal(shareslist)
	shrmux.RUnlock()

	WriteJson(w, http.StatusOK, jb)
}

// APIHANDLER
func shraddApi(w http.ResponseWriter, r *http.Request) {
	// get arguments
	var fpath = r.FormValue("path")
	if len(fpath) == 0 {
		WriteError400(w, ErrArgNoPath, EC_addshrnopath)
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
		WriteJson(w, http.StatusNotFound, &AjaxErr{err, EC_addshrbadpath})
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
func shrdelApi(w http.ResponseWriter, r *http.Request) {
	var ok bool

	// get arguments
	var pref = r.FormValue("pref")
	var path = r.FormValue("path")

	if len(pref) > 0 {
		ok = DelSharePref(pref)
	} else if len(path) > 0 {
		ok = DelSharePath(path)
	} else {
		WriteError400(w, ErrArgNoPref, EC_delshrnopath)
		return
	}

	WriteJson(w, http.StatusOK, ok)
}

// The End.
