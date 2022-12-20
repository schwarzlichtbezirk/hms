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
	Log.Infof("version: %s, builton: %s %s\n", BuildVers, BuildDate, BuildTime)

	var err error

	// get confiruration path
	if ConfigPath, err = DetectConfigPath(); err != nil {
		Log.Fatal(err)
	}
	// load content of Config structure from YAML-file.
	if err = cfg.ReadYaml(cfgfile); err != nil {
		Log.Infoln("error on settings file: " + err.Error())
	}
	// rewrite settings from config file
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
	Log.Infof("config path: %s\n", ConfigPath)

	// get package path
	if PackPath, err = DetectPackPath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("package path: %s\n", PackPath)

	// get cache path
	if CachePath, err = DetectCachePath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("cache path: %s\n", CachePath)

	Log.Infoln("starts")

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
				Log.Infoln("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				Log.Infoln("shutting down by cancel")
			} else {
				Log.Infof("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			Log.Infoln("shutting down by break")
		case <-sigterm:
			Log.Infoln("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()

	// load package with data files
	if err = OpenPackage(); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}

	// init database caches
	if err = InitXorm(); err != nil {
		Log.Fatal("can not init XORM: " + err.Error())
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

	// load previous users states
	if err = usercache.ReadYaml(usrfile); err != nil {
		Log.Infoln("error on users list file: " + err.Error())
	}
	Log.Infof("loaded %d items into users list", len(usercache.list))

	// load profiles with roots, hidden and shares lists
	if err = prflist.ReadYaml(prffile); err != nil {
		Log.Fatal("error on profiles file: " + err.Error())
	}

	// run users scanner for statistics
	go UserScanner()

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

				Log.Infof("start http on %s\n", addr)
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
					Log.Infof("shutdown http on %s: %v\n", addr, err)
				} else {
					Log.Infof("stop http on %s\n", addr)
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

				Log.Infof("start tls on %s\n", addr)
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
					Log.Infof("shutdown tls on %s: %v\n", addr, err)
				} else {
					Log.Infof("stop tls on %s\n", addr)
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
		Log.Infof("hint: Open http://localhost%[1]s page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use http://localhost%[1]s/stat for server state monitoring.\n", suff)
	}
	// wait for exit signal
	<-exitctx.Done()
	Log.Infoln("shutting down begin")

	// wait until all server threads will be stopped.
	exitwg.Wait()
	Log.Infoln("server threads completed")

	// wait until all transactions will be done.
	handwg.Wait()
	Log.Infoln("transactions completed")
}

// Shutdown performs graceful network shutdown.
func Shutdown() {
	var wg errgroup.Group

	wg.Go(func() (err error) {
		if err := usercache.WriteYaml(usrfile); err != nil {
			Log.Infoln("error on users list file: " + err.Error())
		}
		return
	})

	wg.Go(func() (err error) {
		if err := prflist.WriteYaml(prffile); err != nil {
			Log.Infoln("error on profiles list file: " + err.Error())
		}
		return
	})

	wg.Go(func() (err error) {
		var ctx = ImgScanner.Stop()
		<-ctx.Done()
		return
	})

	wg.Go(func() (err error) {
		diskcache.Enum(func(fpath string, cell ExpireCell[DiskFS]) bool {
			cell.Wait.Stop()
			cell.Data.Close()
			return true
		})
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
		if xormEngine != nil {
			xormEngine.Close()
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
	Log.Infoln("shutting down complete.")
}

// The End.
