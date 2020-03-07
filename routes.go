package hms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
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

	// admin
	EC_admindeny = 1

	// page/filecache
	EC_pageabsent = 2
	EC_fileabsent = 3

	// share
	EC_sharegone = 4
	EC_sharenone = 5

	// local
	EC_localnopath = 6

	// reload
	EC_reloadnoreq  = 7
	EC_reloadbadreq = 8
	EC_reloadnodata = 9
	EC_reloadbadprf = 10

	// getlog
	EC_getlogbadnum = 11

	// folder
	EC_folderdeny   = 12
	EC_folderabsent = 13

	// addshr
	EC_addshrnopath  = 14
	EC_addshrbadpath = 15

	// delshr
	EC_delshrnopath = 16

	// thumb
	EC_thumbabsent = 20
	EC_thumbbadcnt = 21

	// tmbchk
	EC_tmbchknoreq  = 30
	EC_tmbchkbadreq = 31
	EC_tmbchknodata = 32

	// tmbscn
	EC_tmbscnnoreq  = 40
	EC_tmbscnbadreq = 41
	EC_tmbscnnodata = 42
)

//////////////////
// Routes table //
//////////////////

// Puts application routes to given router.
func RegisterRoutes(gmux *Router) {

	// UI routes

	var devm = gmux.PathPrefix("/dev").Subrouter()
	devm.Path("/").HandlerFunc(pageHandler("/devm/", "main"))
	gmux.Path("/").HandlerFunc(pageHandler("/relm/", "main"))
	for name := range routedpages {
		devm.Path("/" + name).HandlerFunc(pageHandler("/devm/", name)) // development mode
		gmux.Path("/" + name).HandlerFunc(pageHandler("/relm/", name)) // release mode
	}

	// cached thumbs

	gmux.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// files sharing

	for prefix := range routedpaths {
		gmux.PathPrefix(prefix).HandlerFunc(filecacheHandler)
	}

	gmux.PathPrefix(shareprefix).HandlerFunc(AjaxWrap(shareHandler))
	gmux.PathPrefix("/local").HandlerFunc(AdminWrap(localHandler))

	// API routes

	var api = gmux.PathPrefix("/api").Subrouter()
	api.Path("/ping").HandlerFunc(AjaxWrap(pingApi))
	api.Path("/reload").HandlerFunc(AjaxWrap(reloadApi))
	api.Path("/servinfo").HandlerFunc(AjaxWrap(servinfoApi))
	api.Path("/memusage").HandlerFunc(AjaxWrap(memusageApi))
	api.Path("/getlog").HandlerFunc(AjaxWrap(getlogApi))
	api.Path("/getdrv").HandlerFunc(AdminWrap(getdrvApi))
	api.Path("/folder").HandlerFunc(AjaxWrap(folderApi))
	var shr = api.PathPrefix("/share").Subrouter()
	shr.Path("/lst").HandlerFunc(AjaxWrap(shrlstApi))
	shr.Path("/add").HandlerFunc(AdminWrap(shraddApi))
	shr.Path("/del").HandlerFunc(AdminWrap(shrdelApi))
	var tmb = api.PathPrefix("/tmb").Subrouter()
	tmb.Path("/chk").HandlerFunc(AjaxWrap(tmbchkApi))
	tmb.Path("/scn").HandlerFunc(AjaxWrap(tmbscnApi))
}

func registershares() {
	for pref, path := range sharespref {
		var fi, err = FileStat(path)
		if err != nil {
			Log.Printf("can not create share '%s' on path '%s'", pref, path)
			delete(sharespref, pref)
			continue
		}

		var prop = MakeProp(fi, path)
		prop.SetPref(pref)
		shareslist = append(shareslist, prop)
		sharespath[path] = pref
		Log.Printf("created share '%s' on path '%s'", pref, path)
	}
}

////////////////
// Routes API //
////////////////

// HTTP distribution cache
var filecache = map[string][]byte{}

func LoadFiles(path, prefix string) (count, size uint64, errs []error) {
	var err error
	var files []os.FileInfo
	files, err = ioutil.ReadDir(path)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed path scanning \"%s\" for %s prefix: %s", path, prefix, err.Error()))
		return
	}
	for _, file := range files {
		if file.IsDir() {
			var count1, size1 uint64
			var errs1 []error
			count1, size1, errs1 = LoadFiles(path+file.Name()+"/", prefix+file.Name()+"/")
			count += count1
			size += size1
			errs = append(errs, errs1...)
		} else {
			var content []byte
			content, err = ioutil.ReadFile(path + file.Name())
			if err != nil {
				errs = append(errs, fmt.Errorf("failed read file \"%s\" for %s prefix: %s", path+file.Name(), prefix, err.Error()))
			} else {
				var ext = strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".htm" || ext == ".html" {
					content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")) // remove UTF-8 format BOM header
				}
				filecache[prefix+file.Name()] = content
				count++
				size += uint64(len(content))
			}
		}
	}
	return
}

func LogErrors(errs []error) {
	for _, err := range errs {
		Log.Logln("error", err.Error())
	}
}

// Handler wrapper for AJAX API calls without authorization.
func AjaxWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)
		fn(w, r)
	}
}

// Handler wrapper for API with admin checkup.
func AdminWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)

		if !IsAdmin(r) {
			WriteJson(w, http.StatusForbidden, &AjaxErr{ErrDeny, EC_admindeny})
			return
		}

		fn(w, r)
	}
}

func StripPort(hostport string) string {
	var colon = strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	if i := strings.IndexByte(hostport, ']'); i != -1 {
		return strings.TrimPrefix(hostport[:i], "[")
	}
	return hostport[:colon]
}

func IsLocalhost(host string) bool {
	host = StripPort(host)
	if host == "localhost" {
		return true
	}
	var ip = net.ParseIP(host)
	return ip.IsLoopback()
}

func IsAdmin(r *http.Request) bool {
	return IsLocalhost(r.Host)
}

const (
	serverlabel = "hms-go"
	keepconn    = "keep-alive"
	xframe      = "sameorigin"
	jsoncontent = "application/json;charset=utf-8"
	htmlcontent = "text/html;charset=utf-8"
	csscontent  = "text/css;charset=utf-8"
	jscontent   = "text/javascript;charset=utf-8"
)

var mimeext = map[string]string{
	// Common text content
	".json": jsoncontent,
	".html": htmlcontent,
	".htm":  htmlcontent,
	".css":  csscontent,
	".js":   jscontent,
	".mjs":  jscontent,
	".map":  jsoncontent,
	".txt":  "text/plain;charset=utf-8",
	".pdf":  "application/pdf",
	// Image types
	".tga":  "image/x-tga",
	".bmp":  "image/bmp",
	".dib":  "image/bmp",
	".gif":  "image/gif",
	".png":  "image/png",
	".apng": "image/apng",
	".jpg":  "image/jpeg",
	".jpe":  "image/jpeg",
	".jpeg": "image/jpeg",
	".jfif": "image/jpeg",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".cur":  "image/x-icon",
	// Audio types
	".aac": "audio/aac",
	".mp3": "audio/mpeg",
	".wav": "audio/wav",
	".wma": "audio/x-ms-wma",
	".ogg": "audio/ogg",
	// Video types, multimedia containers
	".mpg":  "video/mpeg",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".3gp":  "video/3gpp",
	// Fonts types
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

func WriteStdHeader(w http.ResponseWriter) {
	w.Header().Set("Connection", keepconn)
	w.Header().Set("Server", serverlabel)
	w.Header().Set("X-Frame-Options", xframe)
}

func WriteHtmlHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", htmlcontent)
	WriteStdHeader(w)
}

func WriteJsonHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", jsoncontent)
	WriteStdHeader(w)
}

func WriteErrHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", jsoncontent)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	WriteStdHeader(w)
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
