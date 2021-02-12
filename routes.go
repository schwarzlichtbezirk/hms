package hms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
)

type void = struct{}

// ErrAjax is error object on AJAX API handlers calls.
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

// MarshalJSON is standard JSON interface implementation for errors on Ajax.
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

// Router is local alias for router type.
type Router = mux.Router

// NewRouter is local alias for router creation function.
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

	// page
	EC_pageabsent = 7
	EC_fileabsent = 8

	// file
	EC_filebadaccid = 10
	EC_filenoacc    = 11
	EC_filehidden   = 12
	EC_filenoprop   = 13
	EC_filenofile   = 14
	EC_fileaccess   = 15

	// media
	EC_mediabadaccid = 20
	EC_medianoacc    = 21
	EC_medianopath   = 22
	EC_mediahidden   = 23
	EC_medianoprop   = 24
	EC_medianofile   = 25
	EC_mediaaccess   = 26
	EC_mediaabsent   = 27
	EC_mediabadcnt   = 28

	// thumb
	EC_thumbbadaccid = 30
	EC_thumbnoacc    = 31
	EC_thumbnopath   = 32
	EC_thumbhidden   = 33
	EC_thumbnoprop   = 34
	EC_thumbnofile   = 35
	EC_thumbaccess   = 36
	EC_thumbabsent   = 37
	EC_thumbbadcnt   = 38

	// pubkey
	EC_pubkeyrand = 40

	// signin
	EC_signinnoreq  = 41
	EC_signinbadreq = 42
	EC_signinnodata = 43
	EC_signinnoacc  = 44
	EC_signinpkey   = 45
	EC_signindeny   = 46

	// refrsh
	EC_refrshnoreq  = 50
	EC_refrshbadreq = 51
	EC_refrshnodata = 52
	EC_refrshparse  = 53

	// reload
	EC_reloadload = 60
	EC_reloadtmpl = 61

	// getlog
	EC_getlogbadnum = 62

	// usrlst
	EC_usrlstnoreq  = 63
	EC_usrlstbadreq = 64

	// ishome
	EC_ishomenoreq  = 70
	EC_ishomebadreq = 71
	EC_ishomenoacc  = 72

	// ctgr
	EC_ctgrnoreq  = 80
	EC_ctgrbadreq = 81
	EC_ctgrnodata = 82
	EC_ctgrnopath = 83
	EC_ctgrnocid  = 84
	EC_ctgrnoacc  = 85
	EC_ctgrnoshr  = 86
	EC_ctgrnotcat = 87

	// folder
	EC_foldernoreq  = 90
	EC_folderbadreq = 91
	EC_foldernodata = 92
	EC_foldernoacc  = 93
	EC_foldernopath = 94
	EC_folderaccess = 95
	EC_folderfail   = 96

	// ispath
	EC_ispathnoreq  = 100
	EC_ispathbadreq = 101
	EC_ispathnoacc  = 102
	EC_ispathdeny   = 103

	// tmb/chk
	EC_tmbchknoreq  = 110
	EC_tmbchkbadreq = 111
	EC_tmbchknodata = 112

	// tmb/scn
	EC_tmbscnnoreq  = 113
	EC_tmbscnbadreq = 114
	EC_tmbscnnodata = 115
	EC_tmbscnnoacc  = 116

	// share/lst
	EC_shrlstnoreq  = 120
	EC_shrlstbadreq = 121
	EC_shrlstnoacc  = 122
	EC_shrlstnoshr  = 123

	// share/add
	EC_shraddnoreq  = 130
	EC_shraddbadreq = 131
	EC_shraddnodata = 132
	EC_shraddnoacc  = 133
	EC_shradddeny   = 134
	EC_shraddnopath = 135
	EC_shraddaccess = 136

	// share/del
	EC_shrdelnoreq  = 140
	EC_shrdelbadreq = 141
	EC_shrdelnodata = 142
	EC_shrdelnoacc  = 143
	EC_shrdeldeny   = 144

	// drive/lst
	EC_drvlstnoreq  = 150
	EC_drvlstbadreq = 151
	EC_drvlstnoacc  = 152
	EC_drvlstnoshr  = 153

	// drive/add
	EC_drvaddnoreq  = 160
	EC_drvaddbadreq = 161
	EC_drvaddnodata = 162
	EC_drvaddnoacc  = 163
	EC_drvadddeny   = 164
	EC_drvaddfile   = 165

	// drive/del
	EC_drvdelnoreq  = 170
	EC_drvdelbadreq = 171
	EC_drvdelnodata = 172
	EC_drvdelnoacc  = 173
	EC_drvdeldeny   = 174
	EC_drvdelnopath = 175
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

// Main page routes.
var routemain = []string{
	"/home/", "/ctgr/", "/path/",
}

// Routes aliases.
var routealias = map[string]string{
	"/devm/": devmsuff,
	"/relm/": relmsuff,
	"/plug/": plugsuff,
	"/asst/": asstsuff,
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
	var dacc = devm.PathPrefix("/id{id}/").Subrouter()
	var gacc = gmux.PathPrefix("/id{id}/").Subrouter()
	for _, pref := range routemain {
		dacc.PathPrefix(pref).HandlerFunc(pageHandler(devmsuff, "main"))
		gacc.PathPrefix(pref).HandlerFunc(pageHandler(relmsuff, "main"))
	}

	// wpk-files sharing
	gmux.PathPrefix("/data/").Handler(http.StripPrefix("/data/", http.FileServer(packager)))
	for alias, prefix := range routealias {
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(packager.SubDir(prefix))))
	}

	// file system sharing
	gacc.PathPrefix("/file/").HandlerFunc(AjaxWrap(fileHandler))
	// converted media files
	gacc.PathPrefix("/media/").HandlerFunc(mediaHandler)
	// cached thumbs
	gacc.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// API routes
	var api = gmux.PathPrefix("/api").Subrouter()
	api.Path("/ping").HandlerFunc(AjaxWrap(pingApi))
	api.Path("/purge").HandlerFunc(AuthWrap(purgeApi))
	api.Path("/reload").HandlerFunc(AuthWrap(reloadApi))
	var stc = api.PathPrefix("/stat").Subrouter()
	stc.Path("/srvinf").HandlerFunc(AjaxWrap(srvinfApi))
	stc.Path("/memusg").HandlerFunc(AjaxWrap(memusgApi))
	stc.Path("/cchinf").HandlerFunc(AjaxWrap(cchinfApi))
	stc.Path("/getlog").HandlerFunc(AjaxWrap(getlogApi))
	stc.Path("/usrlst").HandlerFunc(AjaxWrap(usrlstApi))
	var reg = api.PathPrefix("/auth").Subrouter()
	reg.Path("/pubkey").HandlerFunc(AjaxWrap(pubkeyApi))
	reg.Path("/signin").HandlerFunc(AjaxWrap(signinApi))
	reg.Path("/refrsh").HandlerFunc(AjaxWrap(refrshApi))
	var crd = api.PathPrefix("/card").Subrouter()
	crd.Path("/ishome").HandlerFunc(AjaxWrap(ishomeApi))
	crd.Path("/ctgr").HandlerFunc(AjaxWrap(ctgrApi))
	crd.Path("/folder").HandlerFunc(AjaxWrap(folderApi))
	crd.Path("/ispath").HandlerFunc(AuthWrap(ispathApi))
	var tmb = api.PathPrefix("/tmb").Subrouter()
	tmb.Path("/chk").HandlerFunc(AjaxWrap(tmbchkApi))
	tmb.Path("/scn").HandlerFunc(AjaxWrap(tmbscnApi))
	var shr = api.PathPrefix("/share").Subrouter()
	shr.Path("/lst").HandlerFunc(AjaxWrap(shrlstApi))
	shr.Path("/add").HandlerFunc(AuthWrap(shraddApi))
	shr.Path("/del").HandlerFunc(AuthWrap(shrdelApi))
	var drv = api.PathPrefix("/drive").Subrouter()
	drv.Path("/lst").HandlerFunc(AjaxWrap(drvlstApi))
	drv.Path("/add").HandlerFunc(AuthWrap(drvaddApi))
	drv.Path("/del").HandlerFunc(AuthWrap(drvdelApi))
}

////////////////
// Routes API //
////////////////

// Handler wrapper for AJAX API calls without authorization.
func AjaxWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			userajax <- r
		}()
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

// WriteStdHeader setup common response headers.
func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Server", serverlabel)
	w.Header().Set("X-Frame-Options", "sameorigin")
}

// WriteHTMLHeader setup standard response headers for message with HTML content.
func WriteHTMLHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", htmlcontent)
}

// WriteJSONHeader setup standard response headers for message with JSON content.
func WriteJSONHeader(w http.ResponseWriter) {
	WriteStdHeader(w)
	w.Header().Set("Content-Type", jsoncontent)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// WriteJSON writes to response given status code and marshaled body.
func WriteJSON(w http.ResponseWriter, status int, body interface{}) {
	WriteJSONHeader(w)

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

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, body interface{}) {
	WriteJSON(w, http.StatusOK, body)
}

// WriteError puts to response given error status code and ErrAjax formed by given error object.
func WriteError(w http.ResponseWriter, status int, err error, code int) {
	WriteJSONHeader(w)
	w.WriteHeader(status)
	var b, _ = json.Marshal(&ErrAjax{err, code})
	w.Write(b)
}

// WriteError400 puts to response 400 status code and ErrAjax formed by given error object.
func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteError(w, http.StatusBadRequest, err, code)
}

// WriteError500 puts to response 500 status code and ErrAjax formed by given error object.
func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteError(w, http.StatusInternalServerError, err, code)
}

// The End.
