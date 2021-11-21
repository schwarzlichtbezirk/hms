package hms

import (
	"bytes"
	"context"
	"crypto/tls"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
	"golang.org/x/crypto/acme/autocert"
)

var (
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all service goroutines
	exitwg sync.WaitGroup
)

// Package root dir.
var packager wpk.Packager

// Log is global static ring logger object.
var Log = NewLogger(os.Stderr, LstdFlags, 300)

///////////////////////////////
// Startup opening functions //
///////////////////////////////

// openimage opens hms-package.
func openimage() (pack wpk.Packager, err error) {
	var exepath = filepath.Dir(os.Args[0])
	if cfg.AutoCert {
		return mmap.OpenImage(path.Join(exepath, cfg.WPKName))
	} else {
		return bulk.OpenImage(path.Join(exepath, cfg.WPKName))
	}
}

// hot templates reload, during server running
func loadtemplates() (err error) {
	var ts, tc *template.Template
	var load = func(tb *template.Template, pattern string) {
		var tpl []string
		if tpl, err = packager.Glob(pattern); err != nil {
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

// WaitInterrupt returns shutdown signal was recivied and cancels some context.
func WaitInterrupt(cancel context.CancelFunc) {
	// Make exit signal on function exit.
	defer cancel()

	var sigint = make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
	// SIGKILL, SIGQUIT will not be caught.
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// Block until we receive our signal.
	<-sigint
	Log.Println("shutting down by break")
}

// Init performs global data initialisation. Loads configuration files, initializes file cache.
func Init() {
	var err error

	// create context and wait the break
	exitctx, exitfn = context.WithCancel(context.Background())
	go WaitInterrupt(exitfn)

	// get confiruration path
	if ConfigPath, err = DetectConfigPath(); err != nil {
		log.Fatal(err)
	}
	log.Printf("config path: %s\n", ConfigPath)

	// load settings files
	if err = cfg.Load(path.Join(ConfigPath, "settings.yaml")); err != nil {
		Log.Println("error on settings file: " + err.Error())
	}

	// load package with data files
	if packager, err = openimage(); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}
	Log.Printf("package '%s': cached %d files on %d bytes", cfg.WPKName, len(packager.NFTO()), packager.DataSize())

	// insert components templates into pages
	if err = loadtemplates(); err != nil {
		Log.Fatal(err)
	}

	// build caches with given sizes from settings
	initcaches()

	if err = pathcache.Load(path.Join(ConfigPath, "pathcache.yaml")); err != nil {
		Log.Println("error on path cache file: " + err.Error())
		Log.Println("loading of directories cache and users list were missed for a reason path cache loading failure")
	} else {
		// load directories file groups
		Log.Printf("loaded %d items into path cache", len(pathcache.keypath))
		if err = dircache.Load(path.Join(ConfigPath, "dircache.yaml")); err != nil {
			Log.Println("error on directories cache file: " + err.Error())
		}
		Log.Printf("loaded %d items into directories cache", len(dircache.keydir))

		// load previous users states
		if err = usercache.Load(path.Join(ConfigPath, "userlist.yaml")); err != nil {
			Log.Println("error on users list file: " + err.Error())
		}
		Log.Printf("loaded %d items into users list", len(usercache.list))
	}

	// load profiles with roots, hidden and shares lists
	if err = prflist.Load(path.Join(ConfigPath, "profiles.yaml")); err != nil {
		Log.Fatal("error on profiles file: " + err.Error())
	}

	// run users scanner for statistics
	go UserScanner()

	// EXIF parsers
	exifparsers()
}

// Run launches server listeners.
func Run(gmux *Router) {
	for _, addr := range cfg.PortHTTP {
		var addr = addr // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			Log.Printf("start http on %s\n", addr)
			var server = &http.Server{
				Addr:              addr,
				Handler:           gmux,
				ReadTimeout:       cfg.ReadTimeout,
				ReadHeaderTimeout: cfg.ReadHeaderTimeout,
				WriteTimeout:      cfg.WriteTimeout,
				IdleTimeout:       cfg.IdleTimeout,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					Log.Fatalf("failed to serve on %s: %v", addr, err)
					return
				}
			}()

			// wait for exit signal
			<-exitctx.Done()

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
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
					Cache:  autocert.DirCache(path.Join(ConfigPath, "cert")),
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
				ReadTimeout:       cfg.ReadTimeout,
				ReadHeaderTimeout: cfg.ReadHeaderTimeout,
				WriteTimeout:      cfg.WriteTimeout,
				IdleTimeout:       cfg.IdleTimeout,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				if err := server.ListenAndServeTLS(
					path.Join(ConfigPath, "serv.crt"),
					path.Join(ConfigPath, "prvk.pem")); err != http.ErrServerClosed {
					Log.Fatalf("failed to serve on %s: %v", addr, err)
					return
				}
			}()

			// wait for exit signal
			<-exitctx.Done()

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
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

// WaitExit waits until all server threads will be stopped and all transactions will be done.
func WaitExit() {
	// wait for exit signal
	<-exitctx.Done()
	Log.Println("shutting down begin")

	// wait until all server threads will be stopped.
	exitwg.Wait()
	Log.Println("server threads completed")

	// wait until all transactions will be done.
	handwg.Wait()
	Log.Println("transactions completed")
}

// Shutdown performs graceful network shutdown.
func Shutdown() {
	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		if err := pathcache.Save(path.Join(ConfigPath, "pathcache.yaml")); err != nil {
			Log.Println("error on path cache file: " + err.Error())
			Log.Println("saving of directories cache and users list were missed for a reason path cache saving failure")
			return
		}

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := dircache.Save(path.Join(ConfigPath, "dircache.yaml")); err != nil {
				Log.Println("error on directories cache file: " + err.Error())
			}
		}()

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := usercache.Save(path.Join(ConfigPath, "userlist.yaml")); err != nil {
				Log.Println("error on users list file: " + err.Error())
			}
		}()
	}()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		if err := prflist.Save(path.Join(ConfigPath, "profiles.yaml")); err != nil {
			Log.Println("error on profiles list file: " + err.Error())
		}
	}()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		diskcache.Purge()
	}()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		packager.Close()
	}()

	exitwg.Wait()
}

// The End.
