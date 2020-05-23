package hms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/wpk"
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

	// pubkey
	EC_pubkeyrand = 5

	// signin
	EC_signinnoreq  = 10
	EC_signinbadreq = 11
	EC_signinnodata = 12
	EC_signinpkey   = 13
	EC_signindeny   = 14

	// refrsh
	EC_refrshnoreq  = 20
	EC_refrshbadreq = 21
	EC_refrshnodata = 22
	EC_refrshparse  = 23

	// page/filecache
	EC_pageabsent = 30
	EC_fileabsent = 31

	// file
	EC_filebadurl = 32

	// reload
	EC_reloadnoreq  = 34
	EC_reloadbadreq = 35
	EC_reloadnodata = 36
	EC_reloadbadprf = 37

	// getlog
	EC_getlogbadnum = 38

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
	devm.PathPrefix("/path/").HandlerFunc(pageHandler("/devm/", "main"))
	gmux.PathPrefix("/path/").HandlerFunc(pageHandler("/relm/", "main"))

	// cached thumbs

	gmux.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// files sharing

	for prefix := range routedpaths {
		gmux.PathPrefix(prefix).HandlerFunc(filecacheHandler)
	}
	gmux.Path("/pack").HandlerFunc(packageHandler)
	gmux.PathPrefix("/data").Handler(http.StripPrefix("/data", http.FileServer(&datapack)))
	gmux.PathPrefix("/file").HandlerFunc(AjaxWrap(fileHandler))

	// API routes

	var api = gmux.PathPrefix("/api").Subrouter()
	api.Path("/ping").HandlerFunc(AjaxWrap(pingApi))
	api.Path("/reload").HandlerFunc(AjaxWrap(reloadApi))
	api.Path("/srvinf").HandlerFunc(AjaxWrap(srvinfApi))
	api.Path("/memusg").HandlerFunc(AjaxWrap(memusgApi))
	api.Path("/getlog").HandlerFunc(AjaxWrap(getlogApi))
	api.Path("/pubkey").HandlerFunc(AjaxWrap(pubkeyApi))
	api.Path("/signin").HandlerFunc(AjaxWrap(signinApi))
	api.Path("/refrsh").HandlerFunc(AjaxWrap(refrshApi))
	api.Path("/getdrv").HandlerFunc(AuthWrap(getdrvApi))
	api.Path("/folder").HandlerFunc(AjaxWrap(folderApi))
	api.Path("/purge").HandlerFunc(AuthWrap(purgeApi))
	var shr = api.PathPrefix("/share").Subrouter()
	shr.Path("/lst").HandlerFunc(AjaxWrap(shrlstApi))
	shr.Path("/add").HandlerFunc(AuthWrap(shraddApi))
	shr.Path("/del").HandlerFunc(AuthWrap(shrdelApi))
	var tmb = api.PathPrefix("/tmb").Subrouter()
	tmb.Path("/chk").HandlerFunc(AjaxWrap(tmbchkApi))
	tmb.Path("/scn").HandlerFunc(AjaxWrap(tmbscnApi))
}

func registershares() {
	for pref, path := range sharespref {
		var fi, err = os.Stat(path)
		if err != nil {
			Log.Printf("can not create share '%s' on path '%s'", pref, path)
			delete(sharespref, pref)
			continue
		}

		var prop = MakeProp(path, fi)
		prop.SetPref(pref)
		propcache.Set(path, prop)
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
	if files, err = ioutil.ReadDir(path); err != nil {
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

type Package struct {
	wpk.Package
	body []byte
}

func (pack *Package) Open(fname string) (http.File, error) {
	var key = strings.TrimPrefix(strings.ToLower(filepath.ToSlash(fname)), "/")
	if key == "" {
		return wpk.NewDir(key, &pack.Package), nil
	}
	var tags, is = pack.Tags[key]
	if !is {
		key += "/"
		for k := range pack.Tags {
			if strings.HasPrefix(k, key) {
				return wpk.NewDir(key, &pack.Package), nil
			}
		}
		return nil, ErrNotFound
	}
	var fid = tags.FID()
	var rec = pack.FAT[fid]
	return &wpk.File{
		Reader: *bytes.NewReader(pack.body[uint64(rec.Offset) : uint64(rec.Offset)+uint64(rec.Size)]),
		Tagset: tags,
		Pack:   &pack.Package,
	}, nil
}

var datapack Package

func LoadPackage() {
	var err error

	if datapack.body, err = ioutil.ReadFile(destpath + "hms.wpk"); err != nil {
		Log.Fatal("can not read wpk-package: " + err.Error())
	}

	if err = datapack.Load(bytes.NewReader(datapack.body)); err != nil {
		Log.Fatal("can not open wpk-package: " + err.Error())
	}

	Log.Printf("cached %d files to %d aliases on sum %d bytes", len(datapack.FAT), len(datapack.Tags), datapack.DataSize())
}

// Handler wrapper for AJAX API calls without authorization.
func AjaxWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)
		fn(w, r)
	}
}

var serverlabel string

func MakeServerLabel(label, version string) {
	serverlabel = fmt.Sprintf("%s/%s (%s)", label, version, runtime.GOOS)
}

const (
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
	".aac":  "audio/aac",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".wma":  "audio/x-ms-wma",
	".ogg":  "audio/ogg",
	".flac": "audio/x-flac",
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
