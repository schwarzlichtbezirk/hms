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
	"path"
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

var (
	// channel to indicate about server shutdown
	exitchan chan struct{}
	// wait group for all server goroutines
	exitwg sync.WaitGroup
)

// Package root dir.
var datapack *wpk.Package
var packager wpk.Packager

// Log is global static ring logger object.
var Log = NewLogger(os.Stderr, LstdFlags, 300)

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
		var tpl []string
		if tpl, err = datapack.Glob(pattern); err != nil {
			return
		}
		for _, key := range tpl {
			var bcnt []byte
			if bcnt, err = packager.ReadFile(key); err != nil {
				return
			}
			var content = strings.TrimPrefix(string(bcnt), utf8bom) // remove UTF-8 format BOM header
			if _, err = tb.New(key).Parse(content); err != nil {
				return
			}
		}
	}

	ts = template.New("storage").Delims("[=[", "]=]")
	if load(ts, path.Join(tmplsuff, "*.html")); err != nil {
		return
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(devmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(devmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(relmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(relmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}
	return
}

//////////////////////
// Start web server //
//////////////////////

func pathexists(fpath string) (bool, error) {
	var err error
	if _, err = os.Stat(fpath); err == nil {
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
	var fpath string
	var gopath = filepath.ToSlash(os.Getenv("GOPATH"))
	if !strings.HasSuffix(gopath, "/") {
		gopath += "/"
	}

	// fetch program path
	destpath = path.Dir(filepath.ToSlash(os.Args[0]))

	// fetch configuration path
	if fpath = os.Getenv("APPCONFIGPATH"); fpath == "" {
		fpath = path.Join(destpath, rootsuff)
		if ok, _ := pathexists(fpath); !ok {
			fpath = path.Join(gopath, csrcsuff, confsuff)
			if ok, _ := pathexists(fpath); !ok {
				Log.Fatalf("config folder does not found")
			}
		}
	}
	confpath = fpath

	// load settings files
	if err = cfg.Load(path.Join(confpath, "settings.yaml")); err != nil {
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
	if err = packager.OpenWPK(path.Join(destpath, cfg.WPKName)); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}
	Log.Printf("cached %d package files to %d aliases on %d bytes", datapack.RecNumber, datapack.TagNumber, datapack.TagOffset)

	// insert components templates into pages
	if err = loadtemplates(); err != nil {
		Log.Fatal(err)
	}

	// build caches with given sizes from settings
	initcaches()

	if err = pathcache.Load(path.Join(confpath, "pathcache.yaml")); err != nil {
		Log.Println("error on path cache file: " + err.Error())
		Log.Println("loading of directories cache and users list were missed for a reason path cache loading failure")
	} else {
		// load directories file groups
		Log.Printf("loaded %d items into path cache", len(pathcache.keypath))
		if err = dircache.Load(path.Join(confpath, "dircache.yaml")); err != nil {
			Log.Println("error on directories cache file: " + err.Error())
		}
		Log.Printf("loaded %d items into directories cache", len(dircache.keydir))

		// load previous users states
		if err = usercache.Load(path.Join(confpath, "userlist.yaml")); err != nil {
			Log.Println("error on users list file: " + err.Error())
		}
		Log.Printf("loaded %d items into users list", len(usercache.list))
	}

	// load profiles with roots, hidden and shares lists
	if err = prflist.Load(path.Join(confpath, "profiles.yaml")); err != nil {
		Log.Fatal("error on profiles file: " + err.Error())
	}

	// run users scanner for statistics
	go UserScanner()

	// EXIF parsers
	exifparsers()
}

// Run launches server listeners.
func Run(gmux *Router) {
	// inits exit channel
	exitchan = make(chan struct{})

	for _, addr := range cfg.PortHTTP {
		var addr = addr // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			Log.Printf("start http on %s\n", addr)
			var server = &http.Server{
				Addr:              addr,
				Handler:           gmux,
				ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
				ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
				WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
				IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					Log.Fatalf("failed to serve on %s: %v", addr, err)
					return
				}
			}()

			// wait for exit signal
			<-exitchan

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(
				context.Background(),
				time.Duration(cfg.ShutdownTimeout)*time.Second)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				Log.Printf("shutdown http on %s: %v\n", addr, err)
			} else {
				Log.Printf("stop http on %s\n", addr)
			}
		}()
	}

	for _, addr := range cfg.PortTLS {
		var addr = addr // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			Log.Printf("start tls on %s\n", addr)
			var config *tls.Config
			if cfg.AutoCert { // get certificate from letsencrypt.org
				var m = &autocert.Manager{
					Prompt: autocert.AcceptTOS,
					Cache:  autocert.DirCache(path.Join(confpath, "cert")),
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
			var server = &http.Server{
				Addr:              addr,
				Handler:           gmux,
				TLSConfig:         config,
				ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
				ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
				WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
				IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				if err := server.ListenAndServeTLS(
					path.Join(confpath, "serv.crt"),
					path.Join(confpath, "prvk.pem")); err != http.ErrServerClosed {
					Log.Fatalf("failed to serve on %s: %v", addr, err)
					return
				}
			}()

			// wait for exit signal
			<-exitchan

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(
				context.Background(),
				time.Duration(cfg.ShutdownTimeout)*time.Second)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				Log.Printf("shutdown tls on %s: %v\n", addr, err)
			} else {
				Log.Printf("stop tls on %s\n", addr)
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
	close(exitchan)
	exitwg.Wait()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		if err := pathcache.Save(path.Join(confpath, "pathcache.yaml")); err != nil {
			Log.Println("error on path cache file: " + err.Error())
			Log.Println("saving of directories cache and users list were missed for a reason path cache saving failure")
			return
		}

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := dircache.Save(path.Join(confpath, "dircache.yaml")); err != nil {
				Log.Println("error on directories cache file: " + err.Error())
			}
		}()

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := usercache.Save(path.Join(confpath, "userlist.yaml")); err != nil {
				Log.Println("error on users list file: " + err.Error())
			}
		}()
	}()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		if err := prflist.Save(path.Join(confpath, "profiles.yaml")); err != nil {
			Log.Println("error on profiles list file: " + err.Error())
		}
	}()

	packager.Close()

	exitwg.Wait()
}

// The End.
