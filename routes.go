package hms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

// AjaxGetArg fetch and unmarshal request argument.
func AjaxGetArg(r *http.Request, arg interface{}) error {
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err := json.Unmarshal(jb, arg); err != nil {
			return &ErrAjax{err, AECbadjson}
		}
	} else {
		return &ErrAjax{ErrNoJSON, AECnoreq}
	}
	return nil
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
	if body == nil {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		return
	}
	/*if b, ok := body.([]byte); ok {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		w.Write(b)
		return
	}*/
	var b, err = json.Marshal(body)
	if err == nil {
		w.WriteHeader(status)
		WriteJSONHeader(w)
		w.Write(b)
	} else {
		b, _ = json.Marshal(&ErrAjax{err, AECbadbody})
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONHeader(w)
		w.Write(b)
	}
}

// WriteOK puts 200 status code and some data to response.
func WriteOK(w http.ResponseWriter, body interface{}) {
	WriteJSON(w, http.StatusOK, body)
}

// WriteError puts to response given error status code and ErrAjax formed by given error object.
func WriteError(w http.ResponseWriter, status int, err error, code int) {
	WriteJSON(w, status, &ErrAjax{err, code})
}

// WriteError400 puts to response 400 status code and ErrAjax formed by given error object.
func WriteError400(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusBadRequest, &ErrAjax{err, code})
}

// WriteError500 puts to response 500 status code and ErrAjax formed by given error object.
func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteJSON(w, http.StatusInternalServerError, &ErrAjax{err, code})
}

// The End.
