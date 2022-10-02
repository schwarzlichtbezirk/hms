package hms

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/gorilla/mux"
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
func (err *jerr) MarshalYAML() (interface{}, error) {
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
	When unix_t `json:"when" yaml:"when" xml:"when"`
	// unique API error code
	Code int `json:"code,omitempty" yaml:"code,omitempty" xml:"code,omitempty"`
	// URL with problem detailed description
	Info string `json:"info,omitempty" yaml:"info,omitempty" xml:"info,omitempty"`
}

// MakeAjaxErr is AjaxErr simple constructor.
func MakeAjaxErr(what error, code int) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: UnixJSNow(),
		Code: code,
	}
}

// MakeAjaxInfo is AjaxErr constructor with info URL.
func MakeAjaxInfo(what error, code int, info string) *AjaxErr {
	return &AjaxErr{
		What: jerr{what},
		When: UnixJSNow(),
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
			When: UnixJSNow(),
			Code: code,
		},
		Stack: stack,
	}
}

type XmlMap map[string]interface{}

type xmlMapEntry struct {
	XMLName xml.Name
	Value   interface{} `xml:",chardata"`
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

// NewRouter is local alias for router creation function.
var NewRouter = mux.NewRouter

const (
	htmlcontent = "text/html; charset=utf-8"
	csscontent  = "text/css; charset=utf-8"
	jscontent   = "text/javascript; charset=utf-8"
)

// "Server" field for HTTP headers.
var serverlabel = fmt.Sprintf("hms/%s (%s)", buildvers, runtime.GOOS)

// ParseBody fetch and unmarshal request argument.
func ParseBody(w http.ResponseWriter, r *http.Request, arg interface{}) (err error) {
	if jb, _ := io.ReadAll(r.Body); len(jb) > 0 {
		var ctype = r.Header.Get("Content-Type")
		if pos := strings.IndexByte(ctype, ';'); pos != -1 {
			ctype = ctype[:pos]
		}
		if ctype == "application/json" {
			if err = json.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadjson)
				return
			}
		} else if ctype == "application/x-yaml" || ctype == "application/yaml" {
			if err = yaml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadyaml)
				return
			}
		} else if ctype == "application/xml" {
			if err = xml.Unmarshal(jb, arg); err != nil {
				WriteError400(w, r, err, AECbadxml)
				return
			}
		} else {
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

// WriteStdHeader setup common response headers.
func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
}

// WriteHTMLHeader setup standard response headers for message with HTML content.
func WriteHTMLHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("X-Frame-Options", "sameorigin")
	w.Header().Set("Content-Type", htmlcontent)
}

// WriteRet writes to response given status code and marshaled body.
func WriteRet(w http.ResponseWriter, r *http.Request, status int, body interface{}) {
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
	return
}

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, r *http.Request, body interface{}) {
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

// HTTP distribution cache
var pagecache = map[string][]byte{}

// Pages aliases.
var pagealias = map[string]string{
	"main": "main.html",
	"stat": "stat.html",
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

// Transaction locker, locks until handler will be done.
var handwg sync.WaitGroup

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
				var str = string(buf[:stacklen])
				Log.Infoln(str)
				WriteRet(w, r, http.StatusInternalServerError, MakeErrPanic(err, AECpanic, str))
			}
		}()
		go func() {
			userajax <- r
		}()

		// lock before exit check
		handwg.Add(1)
		defer handwg.Done()

		// check on exit during handler is called
		select {
		case <-exitctx.Done():
			return
		default:
		}

		// call the next handler, which can be another middleware in the chain, or the final handler
		next.ServeHTTP(w, r)
	})
}

// RegisterRoutes puts application routes to given router.
func RegisterRoutes(gmux *Router) {
	// main page
	var devm = gmux.PathPrefix("/dev").Subrouter()
	devm.Path("/").HandlerFunc(pageHandler(devmsuff, "main"))
	gmux.Path("/").HandlerFunc(pageHandler(relmsuff, "main"))
	for name := range pagealias {
		devm.Path("/" + name).HandlerFunc(pageHandler(devmsuff, name)) // development mode
		gmux.Path("/" + name).HandlerFunc(pageHandler(relmsuff, name)) // release mode
	}

	// UI routes
	var dacc = devm.PathPrefix("/id{aid:[0-9]+}/").Subrouter()
	var gacc = gmux.PathPrefix("/id{aid:[0-9]+}/").Subrouter()
	dacc.Use(AjaxMiddleware)
	gacc.Use(AjaxMiddleware)
	for _, pref := range routemain {
		dacc.PathPrefix(pref).HandlerFunc(pageHandler(devmsuff, "main"))
		gacc.PathPrefix(pref).HandlerFunc(pageHandler(relmsuff, "main"))
	}

	// wpk-files sharing
	for alias, prefix := range routealias {
		var sub, err = resfs.Sub(prefix)
		if err != nil {
			Log.Fatal(err)
		}
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(http.FS(sub))))
	}

	// file system sharing & converted media files
	gacc.PathPrefix("/file/").HandlerFunc(fileHandler)
	// cached thumbs and tiles
	gacc.Path("/thumb/{puid}").HandlerFunc(thumbHandler)
	gacc.Path("/tile/{puid}/{wdh:[0-9]+}x{hgt:[0-9]+}").HandlerFunc(tileHandler)

	// API routes
	var api = gmux.PathPrefix("/api").Subrouter()
	api.Use(AjaxMiddleware)
	api.Path("/ping").HandlerFunc(pingAPI)
	api.Path("/reload").HandlerFunc(AuthWrap(reloadAPI))
	api.Path("/stat/srvinf").HandlerFunc(srvinfAPI)
	api.Path("/stat/memusg").HandlerFunc(memusgAPI)
	api.Path("/stat/cchinf").HandlerFunc(cchinfAPI)
	api.Path("/stat/getlog").HandlerFunc(getlogAPI)
	api.Path("/stat/usrlst").HandlerFunc(usrlstAPI)
	api.Path("/auth/pubkey").HandlerFunc(pubkeyAPI)
	api.Path("/auth/signin").HandlerFunc(signinAPI)
	api.Path("/auth/refrsh").HandlerFunc(refrshAPI)
	api.Path("/res/ishome").HandlerFunc(ishomeAPI)
	api.Path("/res/folder").HandlerFunc(folderAPI)
	api.Path("/res/prop").HandlerFunc(propAPI)
	api.Path("/res/ispath").HandlerFunc(AuthWrap(ispathAPI))
	api.Path("/tile/chk").HandlerFunc(tilechkAPI)
	api.Path("/tile/scnstart").HandlerFunc(tilescnstartAPI)
	api.Path("/tile/scnbreak").HandlerFunc(tilescnbreakAPI)
	api.Path("/share/add").HandlerFunc(AuthWrap(shraddAPI))
	api.Path("/share/del").HandlerFunc(AuthWrap(shrdelAPI))
	api.Path("/drive/add").HandlerFunc(AuthWrap(drvaddAPI))
	api.Path("/drive/del").HandlerFunc(AuthWrap(drvdelAPI))
	api.Path("/edit/copy").HandlerFunc(AuthWrap(edtcopyAPI))
	api.Path("/edit/rename").HandlerFunc(AuthWrap(edtrenameAPI))
	api.Path("/edit/delete").HandlerFunc(AuthWrap(edtdeleteAPI))
	api.Path("/gps/range").HandlerFunc(AuthWrap(gpsrangeAPI))
}

// The End.
