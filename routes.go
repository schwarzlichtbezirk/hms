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
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func (e *AjaxErr) AjaxErr() string {
	return fmt.Sprintf("error with code %d: %s", e.Code, e.Message)
}

// Local alias for router type.
type Router = mux.Router

// Local alias for router creation function.
var NewRouter = mux.NewRouter

// API error codes
const (
	EC_null = 0

	EC_unauthorized = 1

	// share
	EC_sharegone = 1
	EC_sharenone = 2

	// local
	EC_localunauth = 3
	EC_localnopath = 4

	// ping
	EC_pingbadreq = 5

	// reload
	EC_reloadbadreq     = 6
	EC_reloadbadcontent = 7
	EC_reloadbadprefix  = 8

	// getlog
	EC_getlogbadnum = 9

	// getdrv
	EC_getdrvunauth = 10

	// folder
	EC_folderabsent = 11

	// addshr
	EC_addshrunauth  = 12
	EC_addshrnopath  = 13
	EC_addshrbadpath = 14

	// delshr
	EC_delshrunauth = 15
	EC_delshrnopath = 16
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

	// files sharing

	for prefix := range routedpaths {
		gmux.PathPrefix(prefix).HandlerFunc(filecacheHandler)
	}

	gmux.PathPrefix(shareprefix).HandlerFunc(shareHandler)
	gmux.PathPrefix("/local").HandlerFunc(localHandler)

	// ajax-queries

	gmux.Path("/api/ping").HandlerFunc(AjaxWrap(pingApi))
	gmux.Path("/api/reload").HandlerFunc(AjaxWrap(reloadApi))
	gmux.Path("/api/servinfo").HandlerFunc(AjaxWrap(servinfoApi))
	gmux.Path("/api/memusage").HandlerFunc(AjaxWrap(memusageApi))
	gmux.Path("/api/getlog").HandlerFunc(AjaxWrap(getlogApi))
	gmux.Path("/api/getdrv").HandlerFunc(AjaxWrap(getdrvApi))
	gmux.Path("/api/folder").HandlerFunc(AjaxWrap(folderApi))
	gmux.Path("/api/shared").HandlerFunc(AjaxWrap(sharedApi))
	gmux.Path("/api/addshr").HandlerFunc(AjaxWrap(addshrApi))
	gmux.Path("/api/delshr").HandlerFunc(AjaxWrap(delshrApi))
}

func registershares() {
	for i := 0; i < len(shareslist); i++ {
		var fp = shareslist[i]

		var f, err = os.Open(fp.Path)
		if err != nil { // check up share valid
			Log.Printf("can not create share '%s' on path '%s'", fp.Pref, fp.Path)
			shareslist = append(shareslist[:i], shareslist[i+1:]...)
			i--
		} else {
			f.Close()
			sharespath[fp.Path] = fp
			sharespref[fp.Pref] = fp
			Log.Printf("created share '%s' on path '%s'", fp.Pref, fp.Path)
		}
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

// handler wrapper for AJAX API calls without authorization
func AjaxWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)
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
	".txt":  "text/plain;charset=utf-8",
	".pdf":  "application/pdf",
	// Image types
	".gif":  "image/gif",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	// Audio types
	".aac": "audio/aac",
	".mp3": "audio/mpeg",
	".wav": "audio/wav",
	".wma": "audio/x-ms-wma",
	".ogg": "audio/ogg",
	// Video types
	".mpg": "video/mpeg",
	".mp4": "video/mp4",
	".wmv": "video/x-ms-wmv",
	".flv": "video/x-flv",
	".3gp": "video/3gpp",
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

func WriteError400(w http.ResponseWriter, err string, code int) {
	WriteJson(w, http.StatusBadRequest, &AjaxErr{err, code})
}

func WriteError500(w http.ResponseWriter, err error, code int) {
	WriteJson(w, http.StatusInternalServerError, &AjaxErr{
		err.Error(),
		code,
	})
}

// The End.
