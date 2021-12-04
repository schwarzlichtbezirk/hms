package hms

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"time"
)

const (
	rootsuff = "hms"
	asstsuff = "assets"  // relative path to assets folder
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
	plugsuff = "plugin"  // relative path to third party code
	confsuff = "config"  // relative path to configuration files folder
	tmplsuff = "tmpl"    // relative path to html templates folder
	csrcsuff = "src/github.com/schwarzlichtbezirk/hms"
)

// CfgAuth is authentication settings.
type CfgAuth struct {
	// Access token time to live.
	AccessTTL time.Duration `json:"access-ttl" yaml:"access-ttl"`
	// Refresh token time to live.
	RefreshTTL time.Duration `json:"refresh-ttl" yaml:"refresh-ttl"`
	// Key for access HS-256 JWT-tokens.
	AccessKey string `json:"access-key" yaml:"access-key"`
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string `json:"refresh-key" yaml:"refresh-key"`
}

// CfgServ is web server settings.
type CfgServ struct {
	AutoCert          bool          `json:"auto-cert" yaml:"auto-cert"`
	PortHTTP          []string      `json:"port-http" yaml:"port-http"`
	PortTLS           []string      `json:"port-tls" yaml:"port-tls"`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout"`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout"`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout"`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout"`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes"`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout"`
}

// CfgSpec is settings for application-specific logic.
type CfgSpec struct {
	// Name of wpk-file with program resources.
	WPKName string `json:"wpk-name" yaml:"wpk-name"`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap"`
	// Maximum duration between two ajax-calls to think client is online.
	OnlineTimeout time.Duration `json:"online-timeout" yaml:"online-timeout"`
	// Default profile for user on localhost.
	DefAccID int `json:"default-profile-id" yaml:"default-profile-id"`
	// Maximum size of image to make thumbnail.
	ThumbFileMaxSize int64 `json:"thumb-file-maxsize" yaml:"thumb-file-maxsize"`
	// Stretch big image embedded into mp3-file to fit into standard icon size.
	FitEmbeddedTmb bool `json:"fit-embedded-tmb" yaml:"fit-embedded-tmb"`
	// Initial size of path unique identifiers in bytes, maximum is 10
	// (x1.6 for length of string representation).
	// When the bottom pool arrives to 90%, size increases to next available value.
	PUIDsize int `json:"puid-size" yaml:"puid-size"`
	// Maximum items number in files properties cache.
	PropCacheMaxNum int `json:"prop-cache-maxnum" yaml:"prop-cache-maxnum"`
	// Maximum items number in thumbnails cache.
	ThumbCacheMaxNum int `json:"thumb-cache-maxnum" yaml:"thumb-cache-maxnum"`
	// Maximum items number in converted media files cache.
	MediaCacheMaxNum int `json:"media-cache-maxnum" yaml:"media-cache-maxnum"`
	// Expiration duration to keep opened iso-disk structures in cache from last access to it.
	DiskCacheExpire time.Duration `json:"disk-cache-expire" yaml:"disk-cache-expire"`
}

// Config is common service settings.
type Config struct {
	CfgAuth `json:"authentication" yaml:"authentication"`
	CfgServ `json:"webserver" yaml:"webserver"`
	CfgSpec `json:"specification" yaml:"specification"`
}

// Instance of common service settings.
var cfg = Config{ // inits default values:
	CfgAuth: CfgAuth{
		AccessTTL:  time.Duration(1*24) * time.Hour,
		RefreshTTL: time.Duration(3*24) * time.Hour,
		AccessKey:  "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479",
		RefreshKey: "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie",
	},
	CfgServ: CfgServ{
		AutoCert:          false,
		PortHTTP:          []string{},
		PortTLS:           []string{},
		ReadTimeout:       time.Duration(15) * time.Second,
		ReadHeaderTimeout: time.Duration(15) * time.Second,
		WriteTimeout:      time.Duration(15) * time.Second,
		IdleTimeout:       time.Duration(60) * time.Second,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   time.Duration(15) * time.Second,
	},
	CfgSpec: CfgSpec{
		WPKName:          "hms.wpk",
		WPKmmap:          false,
		OnlineTimeout:    time.Duration(3*60*1000) * time.Millisecond,
		DefAccID:         1,
		ThumbFileMaxSize: 4096*3072*4 + 65536,
		PUIDsize:         3,
		PropCacheMaxNum:  32 * 1024,
		ThumbCacheMaxNum: 2 * 1024,
		MediaCacheMaxNum: 64,
		DiskCacheExpire:  time.Duration(15) * time.Second,
	},
}

const (
	cfgenv  = "HMSCONFIGPATH"
	cfgfile = "settings.yaml"
	cfgbase = "hms"
	srcpath = "src/github.com/schwarzlichtbezirk/hms/config"

	pcfile = "pathcache.yaml"
	dcfile = "dircache.yaml"
	pffile = "profiles.yaml"
	ulfile = "userlist.yaml"
)

// Configuration path given from command line parameter.
var cfgpath = flag.String("c", "", "configuration path")

// ConfigPath determines configuration path, depended on what directory is exist.
var ConfigPath string

// ErrNoCongig is "no configuration path was found" error message.
var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectConfigPath() (retpath string, err error) {
	var ok bool
	var path string
	var exepath = filepath.Dir(os.Args[0])

	// try to get from environment setting
	if path, ok = os.LookupEnv(cfgenv); ok {
		path = envfmt(path)
		// try to get access to full path
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to find relative from executable path
		path = filepath.Join(exepath, path)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = exepath
			return
		}
		Log.Printf("no access to pointed configuration path '%s'\n", path)
	}

	// try to get from command path arguments
	if path = *cfgpath; path != "" {
		// try to get access to full path
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to find relative from executable path
		path = filepath.Join(exepath, path)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
	}

	// try to get from config subdirectory on executable path
	path = filepath.Join(exepath, cfgbase)
	if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
		retpath = path
		return
	}
	// try to find in executable path
	if ok, _ = pathexists(filepath.Join(exepath, cfgfile)); ok {
		retpath = exepath
		return
	}
	// try to find in current path
	if ok, _ = pathexists(cfgfile); ok {
		retpath = "."
		return
	}

	// if GOBIN is present
	if gobin, ok := os.LookupEnv("GOBIN"); ok {
		// try to get from go bin config
		path = filepath.Join(gobin, cfgbase)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			return
		}
		// try to get from go bin root
		if ok, _ = pathexists(filepath.Join(gobin, cfgfile)); ok {
			retpath = gobin
			return
		}
	}

	// if GOPATH is present
	if gopath, ok := os.LookupEnv("GOPATH"); ok {
		// try to get from go bin config
		path = filepath.Join(gopath, "bin", cfgbase)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to get from go bin root
		path = filepath.Join(gopath, "bin")
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to get from source code
		path = filepath.Join(gopath, srcpath)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}

// Package path given from command line parameter.
var wpkpath = flag.String("w", "", "package path")

// PackPath determines package path, depended on what directory is exist.
var PackPath string

// ErrNoPack is "no package path was found" error message.
var ErrNoPack = errors.New("no package path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectPackPath() (retpath string, err error) {
	var ok bool
	var path string
	var exepath = filepath.Dir(os.Args[0])

	// try to get from command path arguments
	if path = *wpkpath; path != "" {
		// try to get access to full path
		if ok, _ = pathexists(filepath.Join(path, cfg.WPKName)); ok {
			retpath = path
			return
		}
		// try to find relative from executable path
		path = filepath.Join(exepath, path)
		if ok, _ = pathexists(filepath.Join(path, cfg.WPKName)); ok {
			retpath = path
			return
		}
	}

	// try to find in executable path
	if ok, _ = pathexists(filepath.Join(exepath, cfg.WPKName)); ok {
		retpath = exepath
		return
	}
	// try to find in current path
	if ok, _ = pathexists(cfg.WPKName); ok {
		retpath = "."
		return
	}
	// try to find at parental of cofiguration path
	path = filepath.Join(ConfigPath, "..")
	if ok, _ = pathexists(filepath.Join(path, cfg.WPKName)); ok {
		retpath = path
		return
	}

	// if GOBIN is present
	if gobin, ok := os.LookupEnv("GOBIN"); ok {
		if ok, _ = pathexists(filepath.Join(gobin, cfg.WPKName)); ok {
			retpath = gobin
			return
		}
	}

	// if GOPATH is present
	if gopath, ok := os.LookupEnv("GOPATH"); ok {
		// try to get from go bin root
		path = filepath.Join(gopath, "bin")
		if ok, _ = pathexists(filepath.Join(path, cfg.WPKName)); ok {
			retpath = path
			return
		}
	}

	// no package was found
	err = ErrNoPack
	return
}

// The End.
