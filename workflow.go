package hms

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/jessevdk/go-flags"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

var (
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all service goroutines
	exitwg sync.WaitGroup
)

//////////////////////
// Start web server //
//////////////////////

// Init performs global data initialization. Loads configuration files, initializes file cache.
func Init() {
	Log.Infof("version: %s, builton: %s", BuildVers, BuildTime)

	var err error

	// get confiruration path
	if ConfigPath, err = DetectConfigPath(); err != nil {
		Log.Fatal(err)
	}
	// load content of Config structure from YAML-file.
	if err = cfg.ReadYaml(cfgfile); err != nil {
		Log.Error("error at settings file: " + err.Error())
	}
	// rewrite settings from config file
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
	Log.Infof("config path: %s", ConfigPath)

	// get package path
	if PackPath, err = DetectPackPath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("package path: %s", PackPath)

	// get cache path
	if CachePath, err = DetectCachePath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("cache path: %s", CachePath)

	Log.Info("starts")

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
				Log.Info("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				Log.Info("shutting down by cancel")
			} else {
				Log.Infof("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			Log.Info("shutting down by break")
		case <-sigterm:
			Log.Info("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()

	// load package with data files
	if err = OpenPackage(); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}

	// init database caches
	if err = InitStorage(); err != nil {
		Log.Fatal("can not init XORM storage: " + err.Error())
	}
	SqlSession(func(session *Session) (res any, err error) {
		var pathcount, _ = session.Count(&PathStore{})
		Log.Infof("found %d items at system path cache", pathcount)
		var dircount, _ = session.Count(&DirStore{})
		Log.Infof("found %d items at directories cache", dircount)
		var exifcount, _ = session.Count(&ExifStore{})
		Log.Infof("found %d items at EXIF cache", exifcount)
		var tagcount, _ = session.Count(&TagStore{})
		Log.Infof("found %d items at ID3-tags cache", tagcount)
		return
	})

	// load path and GPS caches
	if err = LoadPathCache(); err != nil {
		Log.Fatal("path cache loading failure: " + err.Error())
	}
	Log.Infof("loaded %d items into path cache", pathcache.Len())
	if err = LoadGpsCache(); err != nil {
		Log.Fatal("GPS cache loading failure: " + err.Error())
	}
	Log.Infof("loaded %d items into GPS cache", gpscache.Count())

	if err = InitUserlog(); err != nil {
		Log.Fatal("can not init XORM user log: " + err.Error())
	}
	{
		var uacount, _ = xormUserlog.Count(&AgentStore{})
		Log.Infof("user agent count %d items", uacount)
		Log.Infof("clients count %d", maxcid)
		var opencount, _ = xormUserlog.Count(&OpenStore{})
		Log.Infof("resources open count %d items", opencount)
	}

	// load UaMap
	if err = LoadUaMap(); err != nil {
		Log.Fatal("user agent map loading failure: " + err.Error())
	}

	// insert components templates into pages
	if err = LoadTemplates(); err != nil {
		Log.Fatal(err)
	}

	// build sqlite3 caches
	InitCaches()

	// init wpk caches
	if err = InitPackages(); err != nil {
		Log.Fatal(err)
	}

	// load profiles with roots, hidden and shares lists
	if err = prflist.ReadYaml(prffile); err != nil {
		Log.Fatal("error at profiles file: " + err.Error())
	}

	// load white list
	if err = ReadPasslist(passlst); err != nil {
		Log.Fatal("error at white list file: " + err.Error())
	}

	// run thumbnails scanner
	go ImgScanner.Scan()

	// EXIF parsers
	exifparsers()
}

// Run launches server listeners.
func Run(gmux *Router) {
	// helpers for graceful startup to prevent call to uninitialized data
	var httpctx, httpcancel = context.WithCancel(context.Background())

	// webserver start
	go func() {
		defer httpcancel()
		var httpwg sync.WaitGroup

		// starts HTTP listeners
		for _, addr := range cfg.PortHTTP {
			var addr = addr // localize
			httpwg.Add(1)
			exitwg.Add(1)
			go func() {
				defer exitwg.Done()

				Log.Infof("start http on %s", addr)
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
					httpwg.Done()
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
					Log.Infof("shutdown http on %s: %v", addr, err)
				} else {
					Log.Infof("stop http on %s", addr)
				}
			}()
		}

		// starts HTTPS listeners
		for _, addr := range cfg.PortTLS {
			var addr = addr // localize
			httpwg.Add(1)
			exitwg.Add(1)
			go func() {
				defer exitwg.Done()

				Log.Infof("start tls on %s", addr)
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
					httpwg.Done()
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
					Log.Infof("shutdown tls on %s: %v", addr, err)
				} else {
					Log.Infof("stop tls on %s", addr)
				}
			}()
		}

		httpwg.Wait()
	}()

	select {
	case <-httpctx.Done():
		Log.Infof("webserver ready")
	case <-exitctx.Done():
		return
	}
}

// WaitExit waits until all server threads will be stopped and all transactions will be done.
func WaitExit() {
	if len(cfg.PortHTTP) > 0 {
		var suff string
		var has80 bool
		for _, port := range cfg.PortHTTP {
			if port == ":80" {
				has80 = true
				break
			}
		}
		if !has80 {
			suff = cfg.PortHTTP[0]
		}
		Log.Infof("hint: Open http://localhost%[1]s page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use http://localhost%[1]s/stat for server state monitoring.", suff)
	}
	// wait for exit signal
	<-exitctx.Done()
	Log.Info("shutting down begin")

	// wait until all server threads will be stopped.
	exitwg.Wait()
	Log.Info("server threads completed")

	// wait until all transactions will be done.
	handwg.Wait()
	Log.Info("transactions completed")
}

// Shutdown performs graceful network shutdown.
func Shutdown() {
	var wg errgroup.Group

	wg.Go(func() (err error) {
		if err := prflist.WriteYaml(prffile); err != nil {
			Log.Error("error on profiles list file: " + err.Error())
		}
		return
	})

	wg.Go(func() (err error) {
		var ctx = ImgScanner.Stop()
		<-ctx.Done()
		return
	})

	wg.Go(func() (err error) {
		diskcache.ToLimit(0) // close all opened ISO-files
		return
	})

	wg.Go(func() (err error) {
		thumbpkg.Close()
		return
	})

	wg.Go(func() (err error) {
		tilespkg.Close()
		return
	})

	wg.Go(func() (err error) {
		if xormStorage != nil {
			xormStorage.Close()
		}
		return
	})

	wg.Go(func() (err error) {
		ResFS.Close()
		return
	})

	if err := wg.Wait(); err != nil {
		return
	}
	Log.Info("shutting down complete.")
}

// The End.
