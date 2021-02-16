package hms

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
	"golang.org/x/crypto/acme/autocert"
)

// Log is global static ring logger object.
var Log = NewLogger(os.Stderr, LstdFlags, 300)

var (
	httpsrv, tlssrv []*http.Server
)

// Package root dir.
var datapack *wpk.Package
var packager wpk.Packager

///////////////////////////////
// Startup opening functions //
///////////////////////////////

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
			if bcnt, err = packager.Extract(key); err != nil {
				return
			}
			var content = strings.TrimPrefix(string(bcnt), utf8bom) // remove UTF-8 format BOM header
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

// Init performs global data initialisation. Loads configuration files, initializes file cache.
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
	if path = os.Getenv("APPCONFIGPATH"); path == "" {
		path = destpath + rootsuff
		if ok, _ := pathexists(path); !ok {
			path = gopath + csrcsuff + confsuff
			if ok, _ := pathexists(path); !ok {
				Log.Fatalf("config folder does not found")
			}
		}
	}
	confpath = path

	// load settings files
	if err = cfg.Load(confpath + "settings.yaml"); err != nil {
		Log.Println("error on settings file: " + err.Error())
	}

	// load package with data files
	if cfg.AutoCert {
		var pack = mmap.PackDir{Package: &wpk.Package{}}
		datapack = pack.Package
		packager = &pack
	} else {
		var pack = bulk.PackDir{Package: &wpk.Package{}}
		datapack = pack.Package
		packager = &pack
	}
	if err = packager.OpenWPK(destpath + cfg.WPKName); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}
	Log.Printf("cached %d package files to %d aliases on %d bytes", datapack.RecNumber, datapack.TagNumber, datapack.TagOffset)

	// insert components templates into pages
	if err = loadtemplates(); err != nil {
		Log.Fatal(err)
	}

	// build caches with given sizes from settings
	initcaches()

	if err = pathcache.Load(confpath + "pathcache.yaml"); err != nil {
		Log.Println("error on path cache file: " + err.Error())
		Log.Println("loading of directories cache and users list were missed for a reason path cache loading failure")
	} else {
		// load directories file groups
		Log.Printf("loaded %d items into path cache", len(pathcache.keypath))
		if err = dircache.Load(confpath + "dircache.yaml"); err != nil {
			Log.Println("error on directories cache file: " + err.Error())
		}
		Log.Printf("loaded %d items into directories cache", len(dircache.keydir))

		// load previous users states
		if err = usercache.Load(confpath + "userlist.yaml"); err != nil {
			Log.Println("error on users list file: " + err.Error())
		}
		Log.Printf("loaded %d items into users list", len(usercache.list))
	}

	// load profiles with roots, hidden and shares lists
	if err = prflist.Load(confpath + "profiles.yaml"); err != nil {
		Log.Fatal("error on profiles file: " + err.Error())
	}

	// run users scanner for statistics
	go UserScanner()

	// EXIF parsers
	exifparsers()
}

// Run launches server listeners.
func Run(gmux *Router) {
	httpsrv = make([]*http.Server, len(cfg.AddrHTTP))
	for i, addr := range cfg.AddrHTTP {
		var i = i // make valid access in goroutine
		Log.Println("starts http on " + addr)
		var srv = &http.Server{
			Addr:              addr,
			Handler:           gmux,
			ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
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

	tlssrv = make([]*http.Server, len(cfg.AddrTLS))
	for i, addr := range cfg.AddrTLS {
		var i = i // make valid access in goroutine
		Log.Println("starts tls on " + addr)
		var config *tls.Config
		if cfg.AutoCert { // get certificate from letsencrypt.org
			var m = &autocert.Manager{
				Prompt: autocert.AcceptTOS,
				Cache:  autocert.DirCache(confpath + "cert/"),
			}
			config = &tls.Config{
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
			TLSConfig:         config,
			ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
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

// WaitBreak blocks goroutine until Ctrl+C will be pressed.
func WaitBreak() {
	var sigint = make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
	// SIGKILL, SIGQUIT will not be caught.
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-sigint
}

// Shutdown performs graceful network shutdown,
// waits until all server threads will be stopped.
func Shutdown() {
	// Create a deadline to wait for.
	var ctx, cancel = context.WithTimeout(
		context.Background(),
		time.Duration(cfg.ShutdownTimeout)*time.Second)
	defer cancel()

	var wg sync.WaitGroup // perform shutdown in several goroutines

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	for _, srv := range httpsrv {
		var srv = srv // make valid access in goroutine
		if srv == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.SetKeepAlivesEnabled(false)
			if err := srv.Shutdown(ctx); err != nil {
				Log.Printf("TLS server Shutdown: %v", err)
			}
		}()
	}

	wg.Wait()
	Log.Println("web server stopped")

	// Stop users scanner
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(userquit)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pathcache.Save(confpath + "pathcache.yaml"); err != nil {
			Log.Println("error on path cache file: " + err.Error())
			Log.Println("saving of directories cache and users list were missed for a reason path cache saving failure")
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dircache.Save(confpath + "dircache.yaml"); err != nil {
				Log.Println("error on directories cache file: " + err.Error())
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := usercache.Save(confpath + "userlist.yaml"); err != nil {
				Log.Println("error on users list file: " + err.Error())
			}
		}()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := prflist.Save(confpath + "profiles.yaml"); err != nil {
			Log.Println("error on profiles list file: " + err.Error())
		}
	}()

	packager.Close()

	wg.Wait()
}

// The End.
