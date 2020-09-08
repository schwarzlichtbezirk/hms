package hms

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-ini/ini"
	"golang.org/x/crypto/acme/autocert"
)

const (
	rootsuff = "hms/"
	asstsuff = "assets/"  // relative path to assets folder
	devmsuff = "devmode/" // relative path to folder with development mode code files
	relmsuff = "build/"   // relative path to folder with compiled code files
	plugsuff = "plugin/"  // relative path to third party code
	confsuff = "conf/"    // relative path to configuration files folder
	tmplsuff = "tmpl/"    // relative path to html templates folder
	csrcsuff = "src/github.com/schwarzlichtbezirk/hms/"
)

var (
	destpath string // contains program destination path
	confpath string
)

// web server settings
var (
	AddrHTTP          []string
	AddrTLS           []string
	AutoCert          bool = false
	ReadTimeout       int  = 15
	ReadHeaderTimeout int  = 15
	WriteTimeout      int  = 15
	IdleTimeout       int  = 60 // in seconds
	MaxHeaderBytes    int  = 1 << 20
)

var Log = NewLogger(os.Stderr, LstdFlags, 300)

var starttime = time.Now() // save server start time
var httpsrv, tlssrv []*http.Server

// roots list
var roots []string

// patterns for hidden files
var hidden []string

///////////////////////////////
// Startup opening functions //
///////////////////////////////

func opensettings() {
	var cfg, err = ini.Load(confpath + "settings.ini")
	if err != nil {
		Log.Fatal("can not read settings file: " + err.Error())
	}

	var auth = cfg.Section("authentication")
	AuthPass = auth.Key("password").MustString("dag qus fly in the sky")
	AccessTTL = auth.Key("access-ttl").MustInt(1 * 24 * 60 * 60)
	RefreshTTL = auth.Key("refresh-ttl").MustInt(3 * 24 * 60 * 60)
	AccessKey = auth.Key("access-key").MustString("skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479")
	RefreshKey = auth.Key("refresh-key").MustString("zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie")
	ShowSharesUser = auth.Key("show-shares-user").MustBool(true)

	var photo = cfg.Section("photo")
	ThumbMaxFile = photo.Key("thumb-max-file").MustInt64(4096*3072*4 + 65536)

	var ws = cfg.Section("webserver")
	AddrHTTP = ws.Key("addr-http").Strings(",")
	AddrTLS = ws.Key("addr-tls").Strings(",")
	AutoCert = ws.Key("auto-cert").MustBool(false)
	ReadTimeout = ws.Key("read-timeout").MustInt(15)
	ReadHeaderTimeout = ws.Key("read-header-timeout").MustInt(15)
	WriteTimeout = ws.Key("write-timeout").MustInt(15)
	IdleTimeout = ws.Key("idle-timeout").MustInt(60)
	MaxHeaderBytes = ws.Key("max-header-bytes").MustInt(1 << 20)
}

func loadroots() {
	var err error

	var body []byte
	if body, err = ioutil.ReadFile(confpath + "roots.json"); err != nil {
		Log.Fatal("can not read roots list file: " + err.Error())
	}

	var dec = json.NewDecoder(bytes.NewReader(body))
	if err = dec.Decode(&roots); err != nil {
		Log.Fatal("can not decode roots list: " + err.Error())
	}

	// bring all to valid slashes
	for i, root := range roots {
		roots[i] = filepath.ToSlash(root)
	}
}

func loadhidden() {
	var err error

	var body []byte
	if body, err = ioutil.ReadFile(confpath + "hidden.json"); err != nil {
		Log.Fatal("can not read hidden filenames patterns: " + err.Error())
	}

	var dec = json.NewDecoder(bytes.NewReader(body))
	if err = dec.Decode(&hidden); err != nil {
		Log.Fatal("can not decode hidden filenames array: " + err.Error())
	}

	// bring all to lowercase
	for i, path := range hidden {
		hidden[i] = strings.ToLower(filepath.ToSlash(path))
	}
}

func loadshared() {
	var err error

	var body []byte
	if body, err = ioutil.ReadFile(confpath + "shared.json"); err != nil {
		Log.Fatal("can not read shared resources list file: " + err.Error())
	}

	var dec = json.NewDecoder(bytes.NewReader(body))
	if err = dec.Decode(&sharespref); err != nil {
		Log.Fatal("can not decode shared list: " + err.Error())
	}
}

func saveshared() {
	var err error

	var buf bytes.Buffer
	var enc = json.NewEncoder(&buf)
	enc.SetIndent("", "\t")
	if err = enc.Encode(sharespref); err != nil {
		Log.Println("can not encode shared list: " + err.Error())
		return
	}

	if err = ioutil.WriteFile(confpath+"shared.json", buf.Bytes(), 0644); err != nil {
		Log.Println("can not write shared resources list file: " + err.Error())
		return
	}
}

var dict = func(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	var dict = make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		var key, ok = values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// hot templates reload, during server running
func loadtemplates() (err error) {
	var ts, tc *template.Template
	var load = func(tb *template.Template, pattern string) {
		err = datapack.Glob(pattern, func(key string) (err error) {
			var bcnt []byte
			if bcnt, err = datapack.Extract(datapack.Tags[key]); err != nil {
				return
			}
			var content = strings.TrimPrefix(string(bcnt), "\xef\xbb\xbf") // remove UTF-8 format BOM header
			if _, err = tb.New(key).Parse(content); err != nil {
				return
			}
			return
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

//////////////////////
// Start web server //
//////////////////////

func pathexists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Performs global data initialisation. Loads configuration files, initializes file cache.
func Init() {
	var err error
	var path string
	var gopath = filepath.ToSlash(os.Getenv("GOPATH"))
	if !strings.HasSuffix(gopath, "/") {
		gopath += "/"
	}

	// fetch program path
	destpath = filepath.ToSlash(filepath.Dir(os.Args[0]) + "/")

	// fetch configuration path
	path = destpath + rootsuff
	if ok, _ := pathexists(path); !ok {
		path = gopath + csrcsuff + confsuff
		if ok, _ := pathexists(path); !ok {
			Log.Fatalf("config folder does not found")
		}
	}
	confpath = path

	// load settings files
	opensettings()
	loadroots()
	loadhidden()
	loadshared()
	// load package with data files
	if err = datapack.ReadWPK(destpath + "hms.wpk"); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}
	Log.Printf("cached %d files to %d aliases on %d bytes", datapack.RecNumber, datapack.TagNumber, datapack.TagOffset)
	// insert components templates into pages
	if err = loadtemplates(); err != nil {
		Log.Fatal(err)
	}

	registershares()

	// run meters updater
	meterscanner = time.AfterFunc(time.Second, meterupdater)

	// EXIF parsers
	exifparsers()
}

// Launch server listeners.
func Run(gmux *Router) {
	httpsrv = make([]*http.Server, len(AddrHTTP))
	for i, addr := range AddrHTTP {
		var i = i // make valid access in goroutine
		Log.Println("starts http on " + addr)
		var srv = &http.Server{
			Addr:              addr,
			Handler:           gmux,
			ReadTimeout:       time.Duration(ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(IdleTimeout) * time.Second,
			MaxHeaderBytes:    MaxHeaderBytes,
		}
		httpsrv[i] = srv
		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				httpsrv[i] = nil
				Log.Println(err)
				return
			}
		}()
	}

	tlssrv = make([]*http.Server, len(AddrTLS))
	for i, addr := range AddrTLS {
		var i = i // make valid access in goroutine
		Log.Println("starts tls on " + addr)
		var cfg *tls.Config
		if AutoCert { // get certificate from letsencrypt.org
			var m = &autocert.Manager{
				Prompt: autocert.AcceptTOS,
				Cache:  autocert.DirCache(confpath + "cert/"),
			}
			cfg = &tls.Config{
				PreferServerCipherSuites: true,
				CurvePreferences: []tls.CurveID{
					tls.CurveP256,
					tls.X25519,
				},
				GetCertificate: m.GetCertificate,
			}
		}
		var srv = &http.Server{
			Addr:              addr,
			Handler:           gmux,
			TLSConfig:         cfg,
			ReadTimeout:       time.Duration(ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(IdleTimeout) * time.Second,
			MaxHeaderBytes:    MaxHeaderBytes,
		}
		tlssrv[i] = srv
		go func() {
			if err := srv.ListenAndServeTLS(confpath+"serv.crt", confpath+"prvk.pem"); err != http.ErrServerClosed {
				tlssrv[i] = nil
				Log.Println(err)
				return
			}
		}()
	}
}

// Blocks goroutine until Ctrl+C will be pressed.
func WaitBreak() {
	var sigint = make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(sigint, os.Interrupt)

	// Block until we receive our signal.
	<-sigint
}

// Graceful stop network processing, waits until all server threads will be stopped.
func Done() {
	// Stop meters updater
	meterscanner.Stop()

	// Create a deadline to wait for.
	var ctx, cancel = context.WithTimeout(
		context.Background(),
		time.Duration(ReadTimeout)*time.Second)
	defer cancel()

	var srvwg sync.WaitGroup
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	for _, srv := range httpsrv {
		var srv = srv // make valid access in goroutine
		if srv == nil {
			continue
		}
		srvwg.Add(1)
		go func() {
			defer srvwg.Done()
			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				Log.Printf("HTTP server Shutdown: %v", err)
			}
		}()
	}
	for _, srv := range tlssrv {
		var srv = srv // make valid access in goroutine
		if srv == nil {
			continue
		}
		srvwg.Add(1)
		go func() {
			defer srvwg.Done()
			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				Log.Printf("TLS server Shutdown: %v", err)
			}
		}()
	}

	srvwg.Add(1)
	go func() {
		defer srvwg.Done()
		saveshared()
	}()

	datapack.Close()

	srvwg.Wait()
}

// The End.
