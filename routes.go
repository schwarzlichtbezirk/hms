package hms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/wpk/bulk"
)

type void = struct{}

// Error on API handlers
type AjaxErr struct {
	what error
	code int
}

func (e *AjaxErr) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.code, e.what.Error())
}

func (e *AjaxErr) Unwrap() error {
	return e.what
}

func (e *AjaxErr) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		What string `json:"what"`
		When int64  `json:"when"`
		Code int    `json:"code,omitempty"`
	}{
		e.what.Error(),
		UnixJSNow(),
		e.code,
	})
}

// Local alias for router type.
type Router = mux.Router

// Local alias for router creation function.
var NewRouter = mux.NewRouter

// API error codes
const (
	EC_null = 0

	// auth
	EC_noauth     = 1
	EC_tokenless  = 2
	EC_tokenerror = 3
	EC_tokenbad   = 4
	EC_tokennoacc = 5

	// pubkey
	EC_pubkeyrand = 6

	// signin
	EC_signinnoreq  = 10
	EC_signinbadreq = 11
	EC_signinnodata = 12
	EC_signinnoacc  = 13
	EC_signinpkey   = 14
	EC_signindeny   = 15

	// refrsh
	EC_refrshnoreq  = 20
	EC_refrshbadreq = 21
	EC_refrshnodata = 22
	EC_refrshparse  = 23

	// page
	EC_pageabsent = 30
	EC_fileabsent = 31

	// file
	EC_filebadaccid = 32
	EC_filenoacc    = 33
	EC_filedeny     = 34

	// reload
	EC_reloadload = 35
	EC_reloadtmpl = 36

	// getlog
	EC_getlogbadnum = 37

	// folder
	EC_folderdeny = 40
	EC_folderfail = 41

	// addshr
	EC_addshrnopath  = 42
	EC_addshrbadpath = 43

	// delshr
	EC_delshrnopath = 44

	// thumb
	EC_thumbabsent = 50
	EC_thumbbadcnt = 51

	// tmbchk
	EC_tmbchknoreq  = 52
	EC_tmbchkbadreq = 53
	EC_tmbchknodata = 54

	// tmbscn
	EC_tmbscnnoreq  = 55
	EC_tmbscnbadreq = 56
	EC_tmbscnnodata = 57
)

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

// Routes aliases.
var routealias = map[string]string{
	"/devm/": devmsuff,
	"/relm/": relmsuff,
	"/plug/": plugsuff,
	"/asst/": asstsuff,
}

// Package root dir.
var datapack bulk.PackDir

// Puts application routes to given router.
func RegisterRoutes(gmux *Router) {

	// UI routes

	var devm = gmux.PathPrefix("/dev").Subrouter()
	devm.Path("/").HandlerFunc(pageHandler(devmsuff, "main"))
	gmux.Path("/").HandlerFunc(pageHandler(relmsuff, "main"))
	for name := range pagealias {
		devm.Path("/" + name).HandlerFunc(pageHandler(devmsuff, name)) // development mode
		gmux.Path("/" + name).HandlerFunc(pageHandler(relmsuff, name)) // release mode
	}

	var dacc = devm.PathPrefix("/id{id}/").Subrouter()
	var gacc = gmux.PathPrefix("/id{id}/").Subrouter()
	dacc.PathPrefix("/path/").HandlerFunc(pageHandler(devmsuff, "main"))
	gacc.PathPrefix("/path/").HandlerFunc(pageHandler(relmsuff, "main"))

	// cached thumbs

	gmux.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// files sharing

	gmux.PathPrefix("/data/").Handler(http.StripPrefix("/data/", http.FileServer(&datapack)))
	for alias, prefix := range routealias {
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(datapack.SubDir(prefix))))
	}
	gacc.PathPrefix("/file/").HandlerFunc(AjaxWrap(fileHandler))

	// API routes

	var api = gmux.PathPrefix("/api").Subrouter()
	api.Path("/ping").HandlerFunc(AjaxWrap(pingApi))
	api.Path("/purge").HandlerFunc(AuthWrap(purgeApi))
	api.Path("/reload").HandlerFunc(AjaxWrap(reloadApi))
	api.Path("/srvinf").HandlerFunc(AjaxWrap(srvinfApi))
	api.Path("/memusg").HandlerFunc(AjaxWrap(memusgApi))
	api.Path("/getlog").HandlerFunc(AjaxWrap(getlogApi))
	api.Path("/pubkey").HandlerFunc(AjaxWrap(pubkeyApi))
	api.Path("/signin").HandlerFunc(AjaxWrap(signinApi))
	api.Path("/refrsh").HandlerFunc(AjaxWrap(refrshApi))
	api.Path("/getdrv").HandlerFunc(AuthWrap(getdrvApi))
	api.Path("/folder").HandlerFunc(AjaxWrap(folderApi))
	var shr = api.PathPrefix("/share").Subrouter()
	shr.Path("/lst").HandlerFunc(AjaxWrap(shrlstApi))
	shr.Path("/add").HandlerFunc(AuthWrap(shraddApi))
	shr.Path("/del").HandlerFunc(AuthWrap(shrdelApi))
	var tmb = api.PathPrefix("/tmb").Subrouter()
	tmb.Path("/chk").HandlerFunc(AjaxWrap(tmbchkApi))
	tmb.Path("/scn").HandlerFunc(AjaxWrap(tmbscnApi))
}

////////////////
// Routes API //
////////////////

// Handler wrapper for AJAX API calls without authorization.
func AjaxWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)
		fn(w, r)
	}
}

const (
	jsoncontent = "application/json;charset=utf-8"
	htmlcontent = "text/html;charset=utf-8"
	csscontent  = "text/css;charset=utf-8"
	jscontent   = "text/javascript;charset=utf-8"
)

var serverlabel string

func MakeServerLabel(label, version string) {
	serverlabel = fmt.Sprintf("%s/%s (%s)", label, version, runtime.GOOS)
}

func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
	w.Header().Set("X-Frame-Options", "sameorigin")
}

func WriteHtmlHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", htmlcontent)
}

func WriteJsonHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", jsoncontent)
}

func WriteErrHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", jsoncontent)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func WriteJson(w http.ResponseWriter, status int, body interface{}) {
	if status == http.StatusOK {
		WriteJsonHeader(w)
	} else {
		WriteErrHeader(w)
	}
	w.WriteHeader(status)

	if body != nil {
		if b, ok := body.([]byte); ok {
			w.Write(b)
		} else {
			var b, _ = json.Marshal(body)
			w.Write(b)
		}
	}
}

func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteJson(w, http.StatusBadRequest, &AjaxErr{err, code})
}

func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteJson(w, http.StatusInternalServerError, &AjaxErr{err, code})
}

// The End.
