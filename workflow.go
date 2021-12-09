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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
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

var starttime = time.Now() // save server start time

// Package root dir.
var packager wpk.Packager

// Log is global static ring logger object.
var Log = NewLogger(os.Stderr, LstdFlags, 300)

///////////////////////////////
// Startup opening functions //
///////////////////////////////

// openimage opens hms-package.
func openimage() (pack wpk.Packager, err error) {
	if cfg.AutoCert {
		return mmap.OpenImage(path.Join(PackPath, cfg.WPKName))
	} else {
		return bulk.OpenImage(path.Join(PackPath, cfg.WPKName))
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

// Init performs global data initialization. Loads configuration files, initializes file cache.
func Init() {
	Log.Println("starts")

	// create context and wait the break
	exitctx, exitfn = context.WithCancel(context.Background())
	go func() {
		// Make exit signal on function exit.
		defer exitfn()

		var sigint = make(chan os.Signal, 1)
		var sigterm = make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
		// SIGKILL, SIGQUIT will not be caught.
		signal.Notify(sigint, syscall.SIGINT)
		signal.Notify(sigterm, syscall.SIGTERM)
		// Block until we receive our signal.
		select {
		case <-exitctx.Done():
			if errors.Is(exitctx.Err(), context.DeadlineExceeded) {
				Log.Println("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				Log.Println("shutting down by cancel")
			} else {
				Log.Printf("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			Log.Println("shutting down by break")
		case <-sigterm:
			Log.Println("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()

	var err error

	// get confiruration path
	if ConfigPath, err = DetectConfigPath(); err != nil {
		Log.Fatal(err)
	}
	Log.Printf("config path: %s\n", ConfigPath)

	// load content of Config structure from YAML-file.
	if !cfg.NoConfig {
		if err = cfg.Load(cfgfile); err != nil {
			Log.Println("error on settings file: " + err.Error())
		}
		// second iteration, rewrite settings from config file
		if _, err = flags.NewParser(&cfg, flags.PassDoubleDash).Parse(); err != nil {
			panic("no way to here")
		}
	}

	// get package path
	if PackPath, err = DetectPackPath(); err != nil {
		Log.Fatal(err)
	}
	Log.Printf("package path: %s\n", PackPath)

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

	if err = pathcache.Load(pcfile); err != nil {
		Log.Println("error on path cache file: " + err.Error())
		Log.Println("loading of directories cache and users list were missed for a reason path cache loading failure")
	} else {
		// load directories file groups
		Log.Printf("loaded %d items into path cache", len(pathcache.keypath))
		if err = dircache.Load(dcfile); err != nil {
			Log.Println("error on directories cache file: " + err.Error())
		}
		Log.Printf("loaded %d items into directories cache", len(dircache.keydir))

		// load previous users states
		if err = usercache.Load(ulfile); err != nil {
			Log.Println("error on users list file: " + err.Error())
		}
		Log.Printf("loaded %d items into users list", len(usercache.list))
	}

	// load profiles with roots, hidden and shares lists
	if err = prflist.Load(pffile); err != nil {
		Log.Fatal("error on profiles file: " + err.Error())
	}

	// run users scanner for statistics
	go UserScanner()

	// EXIF parsers
	exifparsers()
}

// Run launches server listeners.
func Run(gmux *Router) {
	// starts HTTP servers
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

	// starts HTTPS servers
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

	Log.Println("ready")
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
		if err := pathcache.Save(pcfile); err != nil {
			Log.Println("error on path cache file: " + err.Error())
			Log.Println("saving of directories cache and users list were missed for a reason path cache saving failure")
			return
		}

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := dircache.Save(dcfile); err != nil {
				Log.Println("error on directories cache file: " + err.Error())
			}
		}()

		exitwg.Add(1)
		go func() {
			defer exitwg.Done()
			if err := usercache.Save(ulfile); err != nil {
				Log.Println("error on users list file: " + err.Error())
			}
		}()
	}()

	exitwg.Add(1)
	go func() {
		defer exitwg.Done()
		if err := prflist.Save(pffile); err != nil {
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
	Log.Println("shutting down complete.")
}

// The End.
