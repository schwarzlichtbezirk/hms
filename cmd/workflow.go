package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	srv "github.com/schwarzlichtbezirk/hms/server"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

const (
	cfgfile = "hms.yaml"
	prffile = "profiles.yaml"
	passlst = "passlist.yaml"
)

var (
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all service goroutines
	exitwg sync.WaitGroup
)

var (
	JoinFast = srv.JoinFast
	Cfg      = cfg.Cfg
	Log      = cfg.Log
)

//////////////////////
// Start web server //
//////////////////////

// Init performs global data initialization. Loads configuration files, initializes file cache.
func Init() {
	if cfg.DevMode {
		Log.Infof("*running in developer mode*")
	}
	Log.Infof("version: %s, builton: %s", cfg.BuildVers, cfg.BuildTime)

	var err error

	// get confiruration path
	if cfg.ConfigPath, err = cfg.DetectConfigPath(); err != nil {
		Log.Fatal(err)
	}
	// load content of Config structure from YAML-file.
	if err = srv.CfgReadYaml(cfgfile); err != nil {
		Log.Error("error at settings file: " + err.Error())
	}
	// rewrite settings from config file
	if _, err := flags.Parse(Cfg); err != nil {
		Log.Error("error at command line flags: " + err.Error())
		os.Exit(1)
	}
	Log.Infof("config path: %s", cfg.ConfigPath)

	// get package path
	if cfg.PackPath, err = cfg.DetectPackPath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("package path: %s", cfg.PackPath)

	// get cache path
	if cfg.CachePath, err = cfg.DetectCachePath(); err != nil {
		Log.Fatal(err)
	}
	Log.Infof("cache path: %s", cfg.CachePath)

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
	if err = srv.OpenPackage(); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}

	// init database caches
	if err = srv.InitStorage(); err != nil {
		Log.Fatal("can not init XORM storage: " + err.Error())
	}
	srv.SqlSession(func(session *srv.Session) (res any, err error) {
		var pathcount, _ = session.Count(&srv.PathStore{})
		Log.Infof("found %d items at system path cache", pathcount)
		var dircount, _ = session.Count(&srv.DirStore{})
		Log.Infof("found %d items at directories cache", dircount)
		var exifcount, _ = session.Count(&srv.ExifStore{})
		Log.Infof("found %d items at EXIF cache", exifcount)
		var tagcount, _ = session.Count(&srv.Id3Store{})
		Log.Infof("found %d items at ID3-tags cache", tagcount)
		return
	})

	// load path, directories and GPS caches
	if err = srv.LoadPathCache(); err != nil {
		Log.Fatal("path cache loading failure: " + err.Error())
	}
	if err = srv.LoadGpsCache(); err != nil {
		Log.Fatal("GPS cache loading failure: " + err.Error())
	}

	if err = srv.InitUserlog(); err != nil {
		Log.Fatal("can not init XORM user log: " + err.Error())
	}
	if err = srv.LoadUaMap(); err != nil {
		Log.Fatal("user agent map loading failure: " + err.Error())
	}

	// insert components templates into pages
	if err = srv.LoadTemplates(); err != nil {
		Log.Fatal(err)
	}

	// init wpk caches
	if err = srv.InitPackages(); err != nil {
		Log.Fatal(err)
	}

	// load profiles with roots, hidden and shares lists
	if err = srv.PrfReadYaml(prffile); err != nil {
		Log.Fatal("error at profiles file: " + err.Error())
	}

	// load white list
	if err = srv.ReadPasslist(passlst); err != nil {
		Log.Fatal("error at white list file: " + err.Error())
	}

	// run thumbnails scanner
	go srv.ImgScanner.Scan()
}

// Run starts main application body.
func Run() {
	if Cfg.CacherMode&cfg.CmCacher != 0 {
		RunCacher()
		select {
		case <-exitctx.Done():
			return
		default:
		}
	}
	if Cfg.CacherMode&cfg.CmWebserver != 0 {
		var gmux = mux.NewRouter()
		srv.RegisterRoutes(gmux)
		RunWeb(gmux)
		WaitExit()
		srv.WaitHandlers()
	}
}

// RunWeb launches server listeners.
func RunWeb(gmux *mux.Router) {
	// helpers for graceful startup to prevent call to uninitialized data
	var httpctx, httpcancel = context.WithCancel(context.Background())

	// webserver start
	go func() {
		defer httpcancel()
		var httpwg sync.WaitGroup

		// starts HTTP listeners
		for _, addr := range Cfg.PortHTTP {
			var addr = addr // localize
			httpwg.Add(1)
			exitwg.Add(1)
			go func() {
				defer exitwg.Done()

				Log.Infof("start http on %s", addr)
				var server = &http.Server{
					Addr:              addr,
					Handler:           gmux,
					ReadTimeout:       Cfg.ReadTimeout,
					ReadHeaderTimeout: Cfg.ReadHeaderTimeout,
					WriteTimeout:      Cfg.WriteTimeout,
					IdleTimeout:       Cfg.IdleTimeout,
					MaxHeaderBytes:    Cfg.MaxHeaderBytes,
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
				var ctx, cancel = context.WithTimeout(context.Background(), Cfg.ShutdownTimeout)
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
		for _, addr := range Cfg.PortTLS {
			var addr = addr // localize
			httpwg.Add(1)
			exitwg.Add(1)
			go func() {
				defer exitwg.Done()

				Log.Infof("start tls on %s", addr)
				var config *tls.Config
				if Cfg.UseAutoCert { // get certificate from letsencrypt.org
					var m = &autocert.Manager{
						Prompt:     autocert.AcceptTOS,
						Cache:      autocert.DirCache(JoinFast(cfg.ConfigPath, "cert")),
						Email:      Cfg.Email,
						HostPolicy: autocert.HostWhitelist(Cfg.HostWhitelist...),
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
					ReadTimeout:       Cfg.ReadTimeout,
					ReadHeaderTimeout: Cfg.ReadHeaderTimeout,
					WriteTimeout:      Cfg.WriteTimeout,
					IdleTimeout:       Cfg.IdleTimeout,
					MaxHeaderBytes:    Cfg.MaxHeaderBytes,
				}
				go func() {
					httpwg.Done()
					if err := server.ListenAndServeTLS(
						JoinFast(cfg.ConfigPath, "serv.crt"),
						JoinFast(cfg.ConfigPath, "prvk.pem")); err != http.ErrServerClosed {
						Log.Fatalf("failed to serve on %s: %v", addr, err)
						return
					}
				}()

				// wait for exit signal
				<-exitctx.Done()

				// create a deadline to wait for.
				var ctx, cancel = context.WithTimeout(context.Background(), Cfg.ShutdownTimeout)
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

	if len(Cfg.PortHTTP) > 0 {
		var suff string
		var has80 bool
		for _, port := range Cfg.PortHTTP {
			if port == ":80" {
				has80 = true
				break
			}
		}
		if !has80 {
			suff = Cfg.PortHTTP[0]
		}
		Log.Infof("hint: Open http://localhost%[1]s page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use http://localhost%[1]s/stat for server state monitoring.", suff)
	}
}

// WaitExit waits until all server threads will be stopped and all transactions will be done.
func WaitExit() {
	// wait for exit signal
	<-exitctx.Done()
	Log.Info("shutting down begin")

	// wait until all server threads will be stopped.
	exitwg.Wait()
	Log.Info("server threads completed")
}

// Done performs graceful network shutdown.
func Done() {
	var wg errgroup.Group

	wg.Go(func() (err error) {
		if err := srv.PrfWriteYaml(prffile); err != nil {
			Log.Error("error on profiles list file: " + err.Error())
		}
		return
	})

	wg.Go(func() (err error) {
		var ctx = srv.ImgScanner.Stop()
		<-ctx.Done()
		return
	})

	// close all opened joints
	wg.Go(func() (err error) {
		srv.JP.Clear()
		return
	})

	wg.Go(func() (err error) {
		srv.ClosePackages()
		return
	})

	wg.Go(func() (err error) {
		if srv.XormStorage != nil {
			srv.XormStorage.Close()
		}
		return
	})

	wg.Go(func() (err error) {
		if srv.XormUserlog != nil {
			srv.XormUserlog.Close()
		}
		return
	})

	wg.Go(func() (err error) {
		srv.ResFS.Close()
		return
	})

	if err := wg.Wait(); err != nil {
		return
	}
	Log.Info("shutting down complete.")
}

// The End.
