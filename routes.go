package hms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
)

type void = struct{}

// Error on AJAX API handlers calls.
type ErrAjax struct {
	What error
	Code int
}

func (e *ErrAjax) Error() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.What.Error())
}

func (e *ErrAjax) Unwrap() error {
	return e.What
}

func (e *ErrAjax) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		What string `json:"what"`
		When int64  `json:"when"`
		Code int    `json:"code,omitempty"`
	}{
		e.What.Error(),
		UnixJSNow(),
		e.Code,
	})
}

// Local alias for router type.
type Router = mux.Router

// Local alias for router creation function.
var NewRouter = mux.NewRouter

// API error codes
const (
	EC_null    = 0
	EC_badjson = 1

	// auth
	EC_noauth     = 2
	EC_tokenless  = 3
	EC_tokenerror = 4
	EC_tokenbad   = 5
	EC_tokennoacc = 6

	// pubkey
	EC_pubkeyrand = 7

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
	EC_filebadaccid = 40
	EC_filenoacc    = 41
	EC_filefpanone  = 42
	EC_filefpaadmin = 43

	// thumb
	EC_thumbbadaccid = 50
	EC_thumbnoacc    = 51
	EC_thumbnopath   = 52
	EC_thumbndeny    = 53
	EC_thumbabsent   = 54
	EC_thumbbadcnt   = 55
	EC_thumbnotcnt   = 56

	// reload
	EC_reloadload = 60
	EC_reloadtmpl = 61

	// getlog
	EC_getlogbadnum = 62

	// folder
	EC_foldernoreq    = 70
	EC_folderbadreq   = 71
	EC_foldernoacc    = 72
	EC_foldernopath   = 73
	EC_folderfpanone  = 74
	EC_folderfpaadmin = 75
	EC_folderfail     = 76

	// ispath
	EC_ispathnoreq  = 80
	EC_ispathbadreq = 81
	EC_ispathnoacc  = 82
	EC_ispathdeny   = 83

	// tmb/chk
	EC_tmbchknoreq  = 90
	EC_tmbchkbadreq = 91
	EC_tmbchknodata = 92

	// tmb/scn
	EC_tmbscnnoreq  = 93
	EC_tmbscnbadreq = 94
	EC_tmbscnnodata = 95
	EC_tmbscnnoacc  = 96

	// share/lst
	EC_shrlstnoreq  = 100
	EC_shrlstbadreq = 101
	EC_shrlstnoacc  = 102
	EC_shrlstdeny   = 103

	// share/add
	EC_shraddnoreq   = 110
	EC_shraddbadreq  = 111
	EC_shraddnodata  = 112
	EC_shraddnoacc   = 113
	EC_shradddeny    = 114
	EC_shraddfpanone = 115

	// share/del
	EC_shrdelnoreq  = 120
	EC_shrdelbadreq = 121
	EC_shrdelnodata = 122
	EC_shrdelnoacc  = 123
	EC_shrdeldeny   = 124
	EC_shrdelnohash = 125

	// drive/add
	EC_drvaddnoreq  = 130
	EC_drvaddbadreq = 131
	EC_drvaddnodata = 132
	EC_drvaddnoacc  = 133
	EC_drvadddeny   = 134
	EC_drvaddfile   = 135

	// drive/del
	EC_drvdelnoreq  = 140
	EC_drvdelbadreq = 141
	EC_drvdelnodata = 142
	EC_drvdelnoacc  = 143
	EC_drvdeldeny   = 144
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

// Puts application routes to given router.
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
	var dacc = devm.PathPrefix("/id{id}/").Subrouter()
	var gacc = gmux.PathPrefix("/id{id}/").Subrouter()
	dacc.PathPrefix("/path/").HandlerFunc(pageHandler(devmsuff, "main"))
	gacc.PathPrefix("/path/").HandlerFunc(pageHandler(relmsuff, "main"))

	// wpk-files sharing
	gmux.PathPrefix("/data/").Handler(http.StripPrefix("/data/", http.FileServer(packager)))
	for alias, prefix := range routealias {
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(packager.SubDir(prefix))))
	}

	// file system sharing
	gacc.PathPrefix("/file/").HandlerFunc(AjaxWrap(fileHandler))

	// cached thumbs
	gacc.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// API routes
	var api = gmux.PathPrefix("/api").Subrouter()
	api.Path("/ping").HandlerFunc(AjaxWrap(pingApi))
	api.Path("/purge").HandlerFunc(AuthWrap(purgeApi))
	api.Path("/reload").HandlerFunc(AjaxWrap(reloadApi))
	api.Path("/srvinf").HandlerFunc(AjaxWrap(srvinfApi))
	api.Path("/memusg").HandlerFunc(AjaxWrap(memusgApi))
	api.Path("/cchinf").HandlerFunc(AjaxWrap(cchinfApi))
	api.Path("/getlog").HandlerFunc(AjaxWrap(getlogApi))
	api.Path("/pubkey").HandlerFunc(AjaxWrap(pubkeyApi))
	api.Path("/signin").HandlerFunc(AjaxWrap(signinApi))
	api.Path("/refrsh").HandlerFunc(AjaxWrap(refrshApi))
	api.Path("/folder").HandlerFunc(AjaxWrap(folderApi))
	api.Path("/ispath").HandlerFunc(AuthWrap(ispathApi))
	var tmb = api.PathPrefix("/tmb").Subrouter()
	tmb.Path("/chk").HandlerFunc(AjaxWrap(tmbchkApi))
	tmb.Path("/scn").HandlerFunc(AjaxWrap(tmbscnApi))
	var shr = api.PathPrefix("/share").Subrouter()
	shr.Path("/lst").HandlerFunc(AjaxWrap(shrlstApi))
	shr.Path("/add").HandlerFunc(AuthWrap(shraddApi))
	shr.Path("/del").HandlerFunc(AuthWrap(shrdelApi))
	var drv = api.PathPrefix("/drive").Subrouter()
	drv.Path("/lst").HandlerFunc(AuthWrap(drvlstApi))
	drv.Path("/add").HandlerFunc(AuthWrap(drvaddApi))
	drv.Path("/del").HandlerFunc(AuthWrap(drvdelApi))
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func WriteJson(w http.ResponseWriter, status int, body interface{}) {
	WriteJsonHeader(w)

	if body != nil {
		var b, err = json.Marshal(body)
		if err == nil {
			w.WriteHeader(status)
			w.Write(b)
		} else {
			b, _ = json.Marshal(&ErrAjax{err, EC_badjson})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(b)
		}
	} else {
		w.WriteHeader(status)
	}
}

func WriteOK(w http.ResponseWriter, body interface{}) {
	WriteJson(w, http.StatusOK, body)
}

func WriteError(w http.ResponseWriter, status int, err error, code int) {
	WriteJsonHeader(w)
	w.WriteHeader(status)
	var b, _ = json.Marshal(&ErrAjax{err, code})
	w.Write(b)
}

func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteError(w, http.StatusBadRequest, err, code)
}

func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteError(w, http.StatusInternalServerError, err, code)
}

// The End.
