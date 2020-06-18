package hms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
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

	// page
	EC_pageabsent = 30
	EC_fileabsent = 31

	// reload
	EC_reloadload = 32
	EC_reloadtmpl = 33

	// getlog
	EC_getlogbadnum = 34

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

var pagealias = map[string]string{
	"main": "main.html",
	"stat": "stat.html",
}

// routes aliases
var routealias = map[string]string{
	"/devm/": devmsuff,
	"/relm/": relmsuff,
	"/plug/": plugsuff,
	"/asst/": asstsuff,
}

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
	devm.PathPrefix("/path/").HandlerFunc(pageHandler(devmsuff, "main"))
	gmux.PathPrefix("/path/").HandlerFunc(pageHandler(relmsuff, "main"))

	// cached thumbs

	gmux.PathPrefix("/thumb/").HandlerFunc(thumbHandler)

	// files sharing

	gmux.PathPrefix("/data/").Handler(http.StripPrefix("/data/", http.FileServer(&datapack)))
	for alias, prefix := range routealias {
		gmux.PathPrefix(alias).Handler(http.StripPrefix(alias, http.FileServer(datapack.Dir(prefix))))
	}
	gmux.Path("/pack").HandlerFunc(packageHandler)
	gmux.PathPrefix("/file/").HandlerFunc(AjaxWrap(fileHandler))

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

type Package struct {
	*wpk.Package
	body []byte
	pref string
}

func (pack *Package) Dir(pref string) *Package {
	return &Package{
		pack.Package,
		pack.body,
		pack.pref + wpk.ToKey(pref),
	}
}

func (pack *Package) Extract(tags wpk.Tagset) []byte {
	var offset, size = tags.Record()
	return pack.body[uint64(offset) : uint64(offset)+uint64(size)]
}

func (pack *Package) Open(kpath string) (http.File, error) {
	var key = pack.pref + strings.TrimPrefix(wpk.ToKey(kpath), "/")
	if key == "" {
		return wpk.NewDir(key, pack.Package), nil
	}
	var tags, is = pack.Tags[key]
	if !is {
		key += "/"
		for k := range pack.Tags {
			if strings.HasPrefix(k, key) {
				return wpk.NewDir(key, pack.Package), nil
			}
		}
		return nil, ErrNotFound
	}

	return &wpk.File{
		Reader: *bytes.NewReader(pack.Extract(tags)),
		Tagset: tags,
		Pack:   pack.Package,
	}, nil
}

var datapack = Package{
	Package: &wpk.Package{},
}

func LoadPackage() (err error) {
	if datapack.body, err = ioutil.ReadFile(destpath + "hms.wpk"); err != nil {
		return
	}

	if err = datapack.Load(bytes.NewReader(datapack.body)); err != nil {
		return
	}

	Log.Printf("cached %d files to %d aliases on %d bytes", datapack.RecNumber, datapack.TagNumber, datapack.TagOffset-wpk.PackHdrSize)
	return
}

// hot templates reload, during server running
func LoadTemplates() (err error) {
	var ts, tc *template.Template
	var load = func(tb *template.Template, pattern string) {
		err = datapack.Glob(pattern, func(key string) error {
			var t = tb.New(key)
			var content = string(datapack.Extract(datapack.Tags[key]))
			content = strings.TrimPrefix(content, "\xef\xbb\xbf") // remove UTF-8 format BOM header
			if _, err = t.Parse(content); err != nil {
				return err
			}
			return nil
		})
	}

	ts = template.New("storage").Delims("[=[", "]=]")
	if load(ts, tmplsuff+"*.html"); err != nil {
		return
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, devmsuff+"*.html"); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		if err = tc.ExecuteTemplate(&buf, devmsuff+fname, nil); err != nil {
			return
		}
		pagecache[devmsuff+fname] = buf.Bytes()
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, relmsuff+"*.html"); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		if err = tc.ExecuteTemplate(&buf, relmsuff+fname, nil); err != nil {
			return
		}
		pagecache[relmsuff+fname] = buf.Bytes()
	}
	return
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
