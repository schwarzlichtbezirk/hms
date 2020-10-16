package hms

import (
	"bytes"
	"context"
	"crypto/tls"
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

	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v3"
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

// Web server settings.
type CfgServ struct {
	AutoCert          bool     `json:"auto-cert" yaml:"auto-cert"`
	AddrHTTP          []string `json:"addr-http" yaml:"addr-http"`
	AddrTLS           []string `json:"addr-tls" yaml:"addr-tls"`
	ReadTimeout       int      `json:"read-timeout" yaml:"read-timeout"`
	ReadHeaderTimeout int      `json:"read-header-timeout" yaml:"read-header-timeout"`
	WriteTimeout      int      `json:"write-timeout" yaml:"write-timeout"`
	IdleTimeout       int      `json:"idle-timeout" yaml:"idle-timeout"`
	MaxHeaderBytes    int      `json:"max-header-bytes" yaml:"max-header-bytes"`
}

type CfgSpec struct {
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap"`
	// Default account for user on localhost.
	DefAccID int `json:"default-account-id" yaml:"default-account-id"`
	// Maximum image size to make thumbnail.
	ThumbMaxFile int64 `json:"thumb-max-file" yaml:"thumb-max-file"`
}

// Common server settings.
var cfg = struct {
	CfgAuth `json:"authentication" yaml:"authentication"`
	CfgSpec `json:"specification" yaml:"specification"`
	CfgServ `json:"webserver" yaml:"webserver"`
}{ // inits default values:
	CfgAuth: CfgAuth{
		AccessTTL:      1 * 24 * 60 * 60,
		RefreshTTL:     3 * 24 * 60 * 60,
		AccessKey:      "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479",
		RefreshKey:     "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie",
		ShowSharesUser: true,
	},
	CfgSpec: CfgSpec{
		WPKmmap:      false,
		DefAccID:     0,
		ThumbMaxFile: 4096*3072*4 + 65536,
	},
	CfgServ: CfgServ{
		AutoCert:          false,
		AddrHTTP:          []string{},
		AddrTLS:           []string{},
		ReadTimeout:       15,
		ReadHeaderTimeout: 15,
		WriteTimeout:      15,
		IdleTimeout:       60,
		MaxHeaderBytes:    1 << 20,
	},
}

var Log = NewLogger(os.Stderr, LstdFlags, 300)

var starttime = time.Now() // save server start time
var httpsrv, tlssrv []*http.Server

// Package root dir.
var datapack *wpk.Package
var packager wpk.Packager

///////////////////////////////
// Startup opening functions //
///////////////////////////////

func opensettings() {
	var err error
	var body []byte
	if body, err = ioutil.ReadFile(confpath + "settings.yaml"); err == nil {
		if err = yaml.Unmarshal(body, &cfg); err != nil {
			Log.Fatal("can not decode settings: " + err.Error())
		}
	} else {
		Log.Println("can not read settings: " + err.Error())
	}

	if cfg.AutoCert {
		var pack = mmap.PackDir{Package: &wpk.Package{}}
		datapack = pack.Package
		packager = &pack
	} else {
		var pack = bulk.PackDir{Package: &wpk.Package{}}
		datapack = pack.Package
		packager = &pack
	}
}

func savesettings() {
	var err error

	var body []byte
	if body, err = yaml.Marshal(&cfg); err != nil {
		Log.Println("can not encode settings: " + err.Error())
		return
	}

	if err = ioutil.WriteFile(confpath+"settings.yaml", body, 0644); err != nil {
		Log.Println("can not write settings file: " + err.Error())
		return
	}
}

func loadaccounts() {
	var err error
	var body []byte
	if body, err = ioutil.ReadFile(confpath + "accounts.yaml"); err == nil {
		if err = yaml.Unmarshal(body, &AccList.list); err != nil {
			Log.Fatal("can not decode accounts array: " + err.Error())
		}
	} else {
		Log.Println("can not read accounts: " + err.Error())
	}

	if len(AccList.list) > 0 {
		for _, acc := range AccList.list {
			Log.Printf("loaded account id%d, login='%s'", acc.ID, acc.Login)
			// bring all roots to valid slashes
			for i, path := range acc.Roots {
				acc.Roots[i] = filepath.ToSlash(path)
			}

			// bring all hissen to lowercase
			for i, path := range acc.Hidden {
				acc.Hidden[i] = strings.ToLower(filepath.ToSlash(path))
			}

			// build shares tables
			acc.UpdateShares()
		}

		// check up default account
		if acc := AccList.ByID(cfg.DefAccID); acc != nil {
			if len(acc.Roots) == 0 {
				acc.FindRoots()
			}
		} else {
			Log.Fatal("default account is not found")
		}
	} else {
		var acc = AccList.NewAccount("admin", "dag qus fly in the sky")
		acc.ID = cfg.DefAccID
		Log.Printf("created account id%d, login='%s'", acc.ID, acc.Login)
		acc.SetDefaultHidden()
		acc.FindRoots()
	}
}

func saveaccounts() {
	var err error

	var body []byte
	if body, err = yaml.Marshal(AccList.list); err != nil {
		Log.Println("can not encode accounts list: " + err.Error())
		return
	}

	if err = ioutil.WriteFile(confpath+"accounts.yaml", body, 0644); err != nil {
		Log.Println("can not write accounts list file: " + err.Error())
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
			if bcnt, err = packager.Extract(key); err != nil {
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
	// load accounts with roots, hidden and shares lists
	loadaccounts()
	// load package with data files
	if err = packager.OpenWPK(destpath + "hms.wpk"); err != nil {
		Log.Fatal("can not load wpk-package: " + err.Error())
	}
	Log.Printf("cached %d files to %d aliases on %d bytes", datapack.RecNumber, datapack.TagNumber, datapack.TagOffset)
	// insert components templates into pages
	if err = loadtemplates(); err != nil {
		Log.Fatal(err)
	}

	// run meters updater
	meterscanner = time.AfterFunc(time.Second, meterupdater)

	// EXIF parsers
	exifparsers()
}

// Launch server listeners.
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
		time.Duration(cfg.ReadTimeout)*time.Second)
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
		saveaccounts()
	}()

	packager.Close()

	srvwg.Wait()
}

// The End.
