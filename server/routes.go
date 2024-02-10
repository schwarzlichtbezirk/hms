package hms

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"
)

var json = jsoniter.ConfigFastest

// AjaxErr is error object on AJAX API handlers calls.
type AjaxErr struct {
	// message with problem description
	What jerr `json:"what" yaml:"what" xml:"what"`
	// time of error rising, in milliseconds of UNIX format
	When time.Time `json:"when" yaml:"when" xml:"when"`
	// unique API error code
	Code int `json:"code,omitempty" yaml:"code,omitempty" xml:"code,omitempty"`
	// URL with problem detailed description
	Info string `json:"info,omitempty" yaml:"info,omitempty" xml:"info,omitempty"`
}

// MakeAjaxErr is AjaxErr simple constructor.
func MakeAjaxErr(what error, code int) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: time.Now(),
		Code: code,
	}
}

// MakeAjaxInfo is AjaxErr constructor with info URL.
func MakeAjaxInfo(what error, code int, info string) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: time.Now(),
		Code: code,
		Info: info,
	}
}

func (e *AjaxErr) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.What.Error())
}

func (e *AjaxErr) Unwrap() error {
	return e.What.error
}

// ErrPanic is error object that helps to get stack trace of goroutine within panic rises.
type ErrPanic struct {
	AjaxErr
	Stack string `json:"stack,omitempty"`
}

// MakeErrPanic is ErrPanic constructor.
func MakeErrPanic(what error, code int, stack string) *ErrPanic {
	return &ErrPanic{
		AjaxErr: AjaxErr{
			What: jerr{what},
			When: time.Now(),
			Code: code,
		},
		Stack: stack,
	}
}

////////////////
// Routes API //
////////////////

// HdrRange describes one range chunk of the file to download.
type HdrRange struct {
	Start int64
	End   int64
}

// GetHdrRange returns array of ranges of file to download from request header.
func GetHdrRange(r *http.Request) (ret []HdrRange) {
	for _, hdr := range r.Header["Range"] {
		var chunks = strings.Split(strings.TrimPrefix(hdr, "bytes="), ", ")
		for _, chunk := range chunks {
			if vals := strings.Split(chunk, "-"); len(vals) == 2 {
				var rv HdrRange
				if vals[0] == "" {
					rv.Start = -1
				} else if i64, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
					rv.Start = i64
				}
				if vals[1] == "" {
					rv.End = -1
				} else if i64, err := strconv.ParseInt(vals[1], 10, 64); err == nil {
					rv.End = i64
				}
				ret = append(ret, rv)
			}
		}
	}
	return
}

// HasRangeBegin returns true if request headers have "Range" header
// with range thats starts from beginning of the file.
func HasRangeBegin(r *http.Request) bool {
	var ranges = GetHdrRange(r)
	if len(ranges) == 0 {
		return true
	}
	for _, rv := range ranges {
		if rv.Start == 0 {
			return true
		}
	}
	return false
}

// WriteStdHeader setup common response headers.
func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
}

// WriteHTMLHeader setup standard response headers for message with HTML content.
func WriteHTMLHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("X-Frame-Options", "sameorigin")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

// WriteRet writes to response given status code and marshaled body.
func WriteRet(w http.ResponseWriter, r *http.Request, status int, body any) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", realmBasic)
		w.Header().Set("WWW-Authenticate", realmBearer)
	}
	if body == nil {
		w.WriteHeader(status)
		WriteStdHeader(w)
		return
	}
	var list []string
	if val := r.Header.Get("Accept"); val != "" {
		list = strings.Split(val, ",")
	} else {
		var ctype = r.Header.Get("Content-Type")
		if ctype == "" {
			ctype = "application/json"
		}
		list = []string{ctype}
	}
	var b []byte
	var err error
	for _, ctype := range list {
		if pos := strings.IndexByte(ctype, ';'); pos != -1 {
			ctype = ctype[:pos]
		}
		switch strings.TrimSpace(ctype) {
		case "*/*", "application/json", "text/json":
			WriteStdHeader(w)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if b, err = json.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		case "application/x-yaml", "application/yaml", "application/yml",
			"text/x-yaml", "text/yaml", "text/yml":
			WriteStdHeader(w)
			w.Header().Set("Content-Type", ctype)
			w.WriteHeader(status)
			if b, err = yaml.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		case "application/xml", "text/xml":
			WriteStdHeader(w)
			w.Header().Set("Content-Type", ctype)
			w.WriteHeader(status)
			if b, err = xml.Marshal(body); err != nil {
				break
			}
			w.Write(b)
			return
		}
	}
	WriteStdHeader(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	if err == nil {
		err = ErrBadEnc // no released encoding was found
	}
	b, _ = json.Marshal(MakeAjaxErr(err, SEC_badenc))
	w.Write(b)

	if status >= 500 {
		if aerr, ok := body.(*AjaxErr); ok {
			Log.Errorf("response status: %d, %s", status, aerr.Error())
		} else {
			Log.Errorf("response status: %d, body: %v", status, body)
		}
	}
}

//////////////////
// Routes table //
//////////////////

const (
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
)

// HTTP distribution cache
var pagecache = map[string][]byte{}

// Pages aliases.
var pagealias = map[string]string{
	"/":     "main.html",
	"/stat": "stat.html",
}

// Routes aliases.
var routealias = map[string]string{
	"/fs/":   ".",
	"/devm/": devmsuff,
	"/relm/": relmsuff,
	"/plug/": "plugin",
	"/asst/": "assets",
}

var ResFS wpk.Union // resources packages root dir.

///////////////////////////////
// Startup opening functions //
///////////////////////////////

// OpenPackage opens hms-package.
func OpenPackage() (err error) {
	for _, fname := range Cfg.WPKName {
		var t0 = time.Now()
		var fpath = JoinPath(cfg.PkgPath, fname)
		var pkg = wpk.NewPackage()
		if err = pkg.OpenFile(fpath); err != nil {
			return
		}

		var dpath string
		if pkg.IsSplitted() {
			dpath = wpk.MakeDataPath(fpath)
		} else {
			dpath = fpath
		}

		if Cfg.WPKmmap {
			pkg.Tagger, err = mmap.MakeTagger(dpath)
		} else {
			pkg.Tagger, err = bulk.MakeTagger(dpath)
		}
		PackInfo(fname, pkg, time.Since(t0))
		ResFS.List = append(ResFS.List, pkg)
	}
	return
}

// LoadTemplates is hot templates reload, during server running.
func LoadTemplates() (err error) {
	var ts, tc *template.Template
	var load = func(tb *template.Template, pattern string) {
		var tpl []string
		if tpl, err = ResFS.Glob(pattern); err != nil {
			return
		}
		for _, key := range tpl {
			var bcnt []byte
			if bcnt, err = ResFS.ReadFile(key); err != nil {
				return
			}
			var content = strings.TrimPrefix(string(bcnt), utf8bom) // remove UTF-8 format BOM header
			if _, err = tb.New(key).Parse(content); err != nil {
				return
			}
		}
	}

	ts = template.New("storage").Delims("[=[", "]=]")
	if load(ts, path.Join("tmpl", "*.html")); err != nil {
		return
	}
	if load(ts, path.Join("tmpl", "*", "*.html")); err != nil { // subfolders
		return
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(devmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(devmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(relmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(relmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}
	return
}

// Transaction locker, locks until handler will be done.
var handwg sync.WaitGroup

// WaitHandlers waits until all transactions will be done.
func WaitHandlers() {
	handwg.Wait()
	Log.Info("transactions completed")
}

// AjaxMiddleware is base handler middleware for AJAX API calls.
func AjaxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if what := recover(); what != nil {
				var err error
				switch v := what.(type) {
				case error:
					err = v
				case string:
					err = errors.New(v)
				case fmt.Stringer:
					err = errors.New(v.String())
				default:
					err = errors.New("panic was thrown at handler")
				}
				var buf [2048]byte
				var stacklen = runtime.Stack(buf[:], false)
				var str = B2S(buf[:stacklen])
				Log.Error(str)
				WriteRet(w, r, http.StatusInternalServerError, MakeErrPanic(err, SEC_panic, str))
			}
		}()

		// lock before exit check
		handwg.Add(1)
		defer handwg.Done()

		var (
			cid          uint64
			uaold, uanew uint64
			isold, isnew bool
		)

		var addr, ua = StripPort(r.RemoteAddr), r.UserAgent()
		uanew = CalcUAID(addr, ua)

		// UAID at cookie
		if uaold, _ = GetUAID(r); uaold == 0 {
			http.SetCookie(w, &http.Cookie{
				Name:  "UAID",
				Value: strconv.FormatUint(uanew, 10),
				Path:  "/",
			})
		}

		uamux.Lock()
		if cid, isnew = UaMap[uanew]; !isnew {
			if cid, isold = UaMap[uaold]; !isold {
				maxcid++
				cid = maxcid
			}
			UaMap[uanew] = cid
			go func() {
				if _, err := XormUserlog.InsertOne(&AgentStore{
					UAID: uanew,
					CID:  cid,
					Addr: addr,
					UA:   ua,
					Lang: r.Header.Get("Accept-Language"),
				}); err != nil {
					panic(err.Error())
				}
			}()
		}
		UserOnline[uanew] = time.Now()
		uamux.Unlock()

		// call the next handler, which can be another middleware in the chain, or the final handler
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes puts application routes to given router.
func RegisterRoutes(gmux *mux.Router) {
	// API routes
	var api = gmux.PathPrefix("/api").Subrouter()
	api.Use(AjaxMiddleware)

	//api.Path("/auth/pubkey").HandlerFunc(pubkeyAPI)
	//api.Path("/auth/signin").HandlerFunc(signinAPI)
	//api.Path("/auth/refrsh").HandlerFunc(refrshAPI)
}

// The End.
