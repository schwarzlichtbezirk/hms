package hms

import (
	"bytes"
	"context"
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
)

const (
	rootsuff = "hms/"
	asstsuff = "assets/"   // relative path to assets folder
	devmsuff = "devmode/"  // relative path to folder with development mode code files
	relmsuff = "build/"    // relative path to folder with compiled code files
	plugsuff = "plugin/"   // relative path to third party code
	confsuff = "config/"   // relative path to configuration files folder
	tmplsuff = "template/" // relative path to html templates folder
	dsrcsuff = "/src/github.com/schwarzlichtbezirk/hms/data/"
)

var (
	destpath string // contains program destination path
	rootpath string
	confpath string
	tmplpath string
	devmpath string
)

var routedpages = map[string]string{
	"main": "main.html",
	"stat": "stat.html",
}
var routedpaths = map[string]string{}

// settings
var (
	AddrHTTP          []string
	AddrTLS           []string
	ReadTimeout       int = 15
	ReadHeaderTimeout int = 15
	WriteTimeout      int = 15
	IdleTimeout       int = 60 // in seconds
	MaxHeaderBytes    int = 1 << 20
)

var Log = NewLogger(os.Stderr, LstdFlags, 300)

var starttime = time.Now() // save server start time
var httpsrv, tlssrv []*http.Server

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

	var ws = cfg.Section("webserver")
	AddrHTTP = ws.Key("addr-http").Strings(",")
	AddrTLS = ws.Key("addr-tls").Strings(",")
	ReadTimeout = ws.Key("read-timeout").MustInt(15)
	ReadHeaderTimeout = ws.Key("read-header-timeout").MustInt(15)
	WriteTimeout = ws.Key("write-timeout").MustInt(15)
	IdleTimeout = ws.Key("idle-timeout").MustInt(60)
	MaxHeaderBytes = ws.Key("max-header-bytes").MustInt(1 << 20)

	var photo = cfg.Section("photo")
	PhotoJPEG = photo.Key("photo-jpeg").MustInt64(2097152)
	PhotoWEBP = photo.Key("photo-webp").MustInt64(1572864)
	ThumbMaxFile = photo.Key("thumb-max-file").MustInt64(4096*3072*4 + 16384)
}

func loadshared() {
	var body, err = ioutil.ReadFile(confpath + "shared.json")
	if err != nil {
		Log.Fatal("can not read shared resources list file: " + err.Error())
	}

	var dec = json.NewDecoder(bytes.NewReader(body))
	err = dec.Decode(&sharespref)
	if err != nil {
		Log.Fatal("can not decode shared list: " + err.Error())
	}
}

func saveshared() {
	var err error
	var buf bytes.Buffer
	var enc = json.NewEncoder(&buf)
	enc.SetIndent("", "\t")
	err = enc.Encode(sharespref)
	if err != nil {
		Log.Println("can not encode shared list: " + err.Error())
		return
	}
	err = ioutil.WriteFile(confpath+"shared.json", buf.Bytes(), 0644)
	if err != nil {
		Log.Println("can not write shared resources list file: " + err.Error())
		return
	}
}

func loadhidden() {
	var body, err = ioutil.ReadFile(confpath + "hidden.json")
	if err != nil {
		Log.Fatal("can not read hidden filenames pattens: " + err.Error())
	}

	var dec = json.NewDecoder(bytes.NewReader(body))
	err = dec.Decode(&hidden)
	if err != nil {
		Log.Fatal("can not decode hidden filenames array: " + err.Error())
	}

	// Bring all to lowercase
	for i, path := range hidden {
		hidden[i] = strings.ToLower(path)
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
func loadtemplates() error {
	var ts, tc *template.Template
	var err error

	ts = template.New("storage").Delims("[=[", "]=]")
	_, err = ts.ParseGlob(tmplpath + "*.html")
	if err != nil {
		return err
	}

	tc, err = ts.Clone()
	if err != nil {
		return err
	}
	_, err = tc.ParseGlob(devmpath + "*.html")
	if err != nil {
		return err
	}
	for _, fname := range routedpages {
		var buf bytes.Buffer
		err = tc.ExecuteTemplate(&buf, fname, nil)
		if err != nil {
			return err
		}
		filecache["/devm/"+fname] = buf.Bytes()
	}

	tc, err = ts.Clone()
	if err != nil {
		return err
	}
	_, err = tc.ParseGlob(devmpath + "*.html")
	if err != nil {
		return err
	}
	for _, fname := range routedpages {
		var buf bytes.Buffer
		err = tc.ExecuteTemplate(&buf, fname, nil)
		if err != nil {
			return err
		}
		filecache["/relm/"+fname] = buf.Bytes()
	}
	return nil
}

//////////////////////
// Start web server //
//////////////////////

func pathexists(path string) (bool, error) {
	var _, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Performs global data initialisation. Loads configuration files, initializes file cache.
func Init() {
	// inits program paths
	destpath, _ = filepath.Split(os.Args[0])
	var gopath = os.Getenv("GOPATH")
	var checkpath = func(suff string) (path string) {
		path = destpath + rootsuff + suff
		if ok, _ := pathexists(path); !ok {
			path = gopath + dsrcsuff + suff
			if ok, _ := pathexists(path); !ok {
				if suff != "" {
					Log.Fatalf("data folder \"%s\" does not found", suff)
				} else {
					Log.Fatal("root data folder does not found")
				}
			}
		}
		return
	}
	rootpath = checkpath("")
	confpath = checkpath(confsuff)
	tmplpath = checkpath(tmplsuff)
	devmpath = checkpath(devmsuff)
	//var relmpath = checkpath(relmsuff)
	var plugpath = checkpath(plugsuff)
	var asstpath = checkpath(asstsuff)

	var err error

	// open settings
	opensettings()
	loadshared()
	loadhidden()
	// make paths routes table
	routedpaths = map[string]string{
		"/devm/": devmpath,
		"/relm/": devmpath, /*relmpath*/ // TODO: put release mode when it will be ready
		"/plug/": plugpath,
		"/asst/": asstpath,
	}
	// cache routed files
	for prefix, path := range routedpaths {
		var count, size, errs = LoadFiles(path, prefix)
		LogErrors(errs)
		Log.Printf("cached %d files on %d bytes for %s route", count, size, prefix)
	}

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
		var srv = &http.Server{
			Addr:              addr,
			Handler:           gmux,
			ReadTimeout:       time.Duration(ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(IdleTimeout) * time.Second,
			MaxHeaderBytes:    MaxHeaderBytes,
		}
		tlssrv[i] = srv
		go func() {
			if err := srv.ListenAndServeTLS(rootpath+"serv.crt", rootpath+"pkey.pem"); err != http.ErrServerClosed {
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

	srvwg.Wait()
}

// The End.
