package hms

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/schwarzlichtbezirk/hms/config"

	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
	"gopkg.in/yaml.v3"
)

type void = struct{}

type jerr struct {
	error
}

// Unwrap returns inherited error object.
func (err *jerr) Unwrap() error {
	return err.error
}

// MarshalJSON is standard JSON interface implementation to stream errors on Ajax.
func (err *jerr) MarshalJSON() ([]byte, error) {
	return json.Marshal(err.Error())
}

// MarshalYAML is YAML marshaler interface implementation to stream errors on Ajax.
func (err *jerr) MarshalYAML() (any, error) {
	return err.Error(), nil
}

// MarshalXML is XML marshaler interface implementation to stream errors on Ajax.
func (err *jerr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(err.Error(), start)
}

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

// Unwrap returns inherited error object.
func (e *AjaxErr) Unwrap() error {
	return e.What
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

type XmlMap map[string]any

type xmlMapEntry struct {
	XMLName xml.Name
	Value   any `xml:",chardata"`
}

// MarshalXML marshals the map to XML, with each key in the map being a
// tag and it's corresponding value being it's contents.
//
// See https://stackoverflow.com/questions/30928770/marshall-map-to-xml-in-go
func (m XmlMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m) == 0 {
		return nil
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	for k, v := range m {
		e.Encode(xmlMapEntry{XMLName: xml.Name{Local: k}, Value: v})
	}

	return e.EncodeToken(start.End())
}

// UnmarshalXML unmarshals the XML into a map of string to strings,
// creating a key in the map for each tag and setting it's value to the
// tags contents.
//
// The fact this function is on the pointer of Map is important, so that
// if m is nil it can be initialized, which is often the case if m is
// nested in another xml structurel. This is also why the first thing done
// on the first line is initialize it.
func (m *XmlMap) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*m = XmlMap{}
	for {
		var e xmlMapEntry

		var err = d.Decode(&e)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		(*m)[e.XMLName.Local] = e.Value
	}
	return nil
}

////////////////
// Routes API //
////////////////

// Router is local alias for router type.
type Router = mux.Router

// "Server" field for HTTP headers.
var serverlabel = fmt.Sprintf("hms/%s (%s)", BuildVers, runtime.GOOS)

// ParseBody fetch and unmarshal request argument.
func ParseBody(w http.ResponseWriter, r *http.Request, arg any) (err error) {
	if jb, _ := io.ReadAll(r.Body); len(jb) > 0 {
		var ctype = r.Header.Get("Content-Type")
		if pos := strings.IndexByte(ctype, ';'); pos != -1 {
			ctype = ctype[:pos]
		}
		switch ctype {
		case "application/json", "text/json":
			if err = json.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadjson)
				return
			}
		case "application/x-yaml", "application/yaml", "application/yml",
			"text/x-yaml", "text/yaml", "text/yml":
			if err = yaml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadyaml)
				return
			}
		case "application/xml", "text/xml":
			if err = xml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadxml)
				return
			}
		default:
			WriteError400(w, r, ErrArgUndef, AECargundef)
			return
		}
	} else {
		err = ErrNoJSON
		WriteError400(w, r, err, AECnoreq)
		return
	}
	return
}

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
	b, _ = json.Marshal(MakeAjaxErr(err, AECbadenc))
	w.Write(b)
}

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, r *http.Request, body any) {
	WriteRet(w, r, http.StatusOK, body)
}

// WriteError puts to response given error status code and AjaxErr formed by given error object.
func WriteError(w http.ResponseWriter, r *http.Request, status int, err error, code int) {
	WriteRet(w, r, status, MakeAjaxErr(err, code))
}

// WriteError400 puts to response 400 status code and AjaxErr formed by given error object.
func WriteError400(w http.ResponseWriter, r *http.Request, err error, code int) {
	WriteRet(w, r, http.StatusBadRequest, MakeAjaxErr(err, code))
}

// WriteError500 puts to response 500 status code and AjaxErr formed by given error object.
func WriteError500(w http.ResponseWriter, r *http.Request, err error, code int) {
	WriteRet(w, r, http.StatusInternalServerError, MakeAjaxErr(err, code))
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

// Main page routes.
var routemain = []string{
	"/home/", "/ctgr/", "/path/",
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
		var fpath = path.Join(PackPath, fname)
		var pkg *wpk.Package
		if pkg, err = wpk.OpenPackage(fpath); err != nil {
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
		PackInfo(fname, pkg)
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

const alias_cond = "(cid1=? AND cid2=?) OR (cid1=? AND cid2=?)"

func WaitHandlers() {
	handwg.Wait()
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
				var str = b2s(buf[:stacklen])
				Log.Error(str)
				WriteRet(w, r, http.StatusInternalServerError, MakeErrPanic(err, AECpanic, str))
			}
		}()

		// lock before exit check
		handwg.Add(1)
		defer handwg.Done()

		var (
			cid          ID_t
			uaold, uanew ID_t
			isold, isnew bool
		)

		var addr, ua = StripPort(r.RemoteAddr), r.UserAgent()
		uanew = CalcUAID(addr, ua)

		// UAID at cookie
		if uaold, _ = GetUAID(r); uaold == 0 {
			http.SetCookie(w, &http.Cookie{
				Name:  "UAID",
				Value: strconv.FormatUint(uint64(uanew), 10),
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
func RegisterRoutes(gmux *Router) {
	gmux.Use(AjaxMiddleware)

	// UI pages
	var devm = gmux.PathPrefix("/dev").Subrouter()
	for fpath, fname := range pagealias {
		devm.Path(fpath).HandlerFunc(pageHandler(devmsuff, fname)) // development mode
		gmux.Path(fpath).HandlerFunc(pageHandler(relmsuff, fname)) // release mode
	}

	// profile routes
	var dacc = devm.PathPrefix("/id{aid:[0-9]+}/").Subrouter()
	var gacc = gmux.PathPrefix("/id{aid:[0-9]+}/").Subrouter()
	for _, pref := range routemain {
		dacc.PathPrefix(pref).HandlerFunc(pageHandler(devmsuff, pagealias["/"]))
		gacc.PathPrefix(pref).HandlerFunc(pageHandler(relmsuff, pagealias["/"]))
	}

	// wpk-files sharing
	for alias, prefix := range routealias {
		var sub, err = ResFS.Sub(prefix)
		if err != nil {
			Log.Fatal(err)
		}
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(http.FS(sub))))
	}

	// file system sharing & converted media files
	gacc.PathPrefix("/file/").HandlerFunc(AuthWrap(fileHandler))
	// embedded thumbnails
	gacc.Path("/etmb/{puid}").HandlerFunc(AuthWrap(etmbHandler))
	// cached thumbnails
	gacc.Path("/mtmb/{puid}").HandlerFunc(AuthWrap(mtmbHandler))
	// cached tiles
	gacc.Path("/tile/{puid}/{wdh:[0-9]+}x{hgt:[0-9]+}").HandlerFunc(AuthWrap(tileHandler))

	// API routes
	var api = gmux.PathPrefix("/api").Subrouter()
	var usr = gacc.PathPrefix("/api").Subrouter()
	api.Use(AjaxMiddleware)
	api.Path("/ping").HandlerFunc(pingAPI)
	api.Path("/reload").HandlerFunc(reloadAPI) // authorized only

	api.Path("/stat/srvinf").HandlerFunc(srvinfAPI)
	api.Path("/stat/memusg").HandlerFunc(memusgAPI)
	api.Path("/stat/cchinf").HandlerFunc(cchinfAPI)
	api.Path("/stat/getlog").HandlerFunc(getlogAPI)
	api.Path("/stat/usrlst").HandlerFunc(usrlstAPI)

	api.Path("/auth/pubkey").HandlerFunc(pubkeyAPI)
	api.Path("/auth/signin").HandlerFunc(signinAPI)
	api.Path("/auth/refrsh").HandlerFunc(refrshAPI)

	usr.Path("/res/folder").HandlerFunc(AuthWrap(folderAPI))
	usr.Path("/res/tags").HandlerFunc(AuthWrap(tagsAPI))
	usr.Path("/res/ispath").HandlerFunc(AuthWrap(ispathAPI)) // authorized only

	usr.Path("/tile/chk").HandlerFunc(AuthWrap(tilechkAPI))
	usr.Path("/tile/scnstart").HandlerFunc(AuthWrap(tilescnstartAPI))
	usr.Path("/tile/scnbreak").HandlerFunc(AuthWrap(tilescnbreakAPI))

	usr.Path("/drive/add").HandlerFunc(AuthWrap(drvaddAPI)) // authorized only
	usr.Path("/drive/del").HandlerFunc(AuthWrap(drvdelAPI)) // authorized only

	usr.Path("/cloud/add").HandlerFunc(AuthWrap(cldaddAPI)) // authorized only
	usr.Path("/cloud/del").HandlerFunc(AuthWrap(clddelAPI)) // authorized only

	usr.Path("/share/add").HandlerFunc(AuthWrap(shraddAPI)) // authorized only
	usr.Path("/share/del").HandlerFunc(AuthWrap(shrdelAPI)) // authorized only

	usr.Path("/edit/copy").HandlerFunc(AuthWrap(edtcopyAPI))     // authorized only
	usr.Path("/edit/rename").HandlerFunc(AuthWrap(edtrenameAPI)) // authorized only
	usr.Path("/edit/delete").HandlerFunc(AuthWrap(edtdeleteAPI)) // authorized only

	usr.Path("/gps/range").HandlerFunc(AuthWrap(gpsrangeAPI)) // authorized only
	usr.Path("/gps/scan").HandlerFunc(AuthWrap(gpsscanAPI))
}

// The End.
