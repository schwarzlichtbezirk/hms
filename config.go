package hms

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
)

const (
	rootsuff = "hms"
	asstsuff = "assets"  // relative path to assets folder
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
	plugsuff = "plugin"  // relative path to third party code
	confsuff = "config"  // relative path to configuration files folder
	tmplsuff = "tmpl"    // relative path to html templates folder
)

// CfgCmdLine is command line arguments representation for YAML settings.
type CfgCmdLine struct {
	ConfigPath string `json:"-" yaml:"-" env:"HMSCONFIGPATH" short:"c" long:"cfgpath" description:"Configuration path. Can be full path to config folder, or relative from executable destination."`
	NoConfig   bool   `json:"-" yaml:"-" long:"nocfg" description:"Specifies do not load settings from YAML-settings file, keeps default."`
	PackPath   string `json:"-" yaml:"-" env:"HMSPACKPATH" long:"wpkpath" description:"Determines package path. Can be full path to folder with package, or relative from executable destination."`
}

// CfgJWTAuth is authentication settings.
type CfgJWTAuth struct {
	// Access token time to live.
	AccessTTL time.Duration `json:"access-ttl" yaml:"access-ttl"`
	// Refresh token time to live.
	RefreshTTL time.Duration `json:"refresh-ttl" yaml:"refresh-ttl"`
	// Key for access HS-256 JWT-tokens.
	AccessKey string `json:"access-key" yaml:"access-key"`
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string `json:"refresh-key" yaml:"refresh-key"`
}

// CfgWebServ is web server settings.
type CfgWebServ struct {
	AutoCert          bool          `json:"auto-cert" yaml:"auto-cert" long:"autocert" description:"Indicates to get TLS-certificate from letsencrypt.org service if this value is true. Uses local TLS-certificate otherwise."`
	PortHTTP          []string      `json:"port-http" yaml:"port-http" short:"w" long:"porthttp" description:"List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains."`
	PortTLS           []string      `json:"port-tls" yaml:"port-tls" short:"s" long:"porttls" description:"List of address:port values for encrypted connections. Address is skipped in most common cases, port only remains."`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout" long:"rt" description:"Maximum duration for reading the entire request, including the body."`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout" long:"rht" description:"Amount of time allowed to read request headers."`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout" long:"wt" description:"Maximum duration before timing out writes of the response."`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout" long:"it" description:"Maximum amount of time to wait for the next request when keep-alives are enabled."`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes" long:"mhb" description:"Controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line, in bytes."`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout" long:"st" description:"Maximum duration to wait for graceful shutdown."`
}

// CfgAppConf is settings for application-specific logic.
type CfgAppConf struct {
	// Name of wpk-file with program resources.
	WPKName string `json:"wpk-name" yaml:"wpk-name" long:"wpk" description:"Name of wpk-file with program resources."`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap" long:"mmap" description:"Memory mapping technology for WPK, or load into one solid byte slice otherwise."`
	// Maximum duration between two ajax-calls to think client is online.
	OnlineTimeout time.Duration `json:"online-timeout" yaml:"online-timeout" long:"ot" description:"Maximum duration between two ajax-calls to think client is online."`
	// Default profile identifier for user on localhost.
	DefAccID IdType `json:"default-profile-id" yaml:"default-profile-id" long:"defaid" description:"Default profile identifier for user on localhost."`
	// Maximum size of image to make thumbnail.
	ThumbFileMaxSize int64 `json:"thumb-file-maxsize" yaml:"thumb-file-maxsize" long:"tfms" description:"Maximum size of image to make thumbnail."`
	// Stretch big image embedded into mp3-file to fit into standard icon size.
	FitEmbeddedTmb bool `json:"fit-embedded-tmb" yaml:"fit-embedded-tmb" long:"fet" description:"Stretch big image embedded into mp3-file to fit into standard icon size."`
	// Initial size of path unique identifiers in bytes, maximum is 10
	// (x1.6 for length of string representation).
	// When the bottom pool arrives to 90%, size increases to next available value.
	PUIDsize int `json:"puid-size" yaml:"puid-size" long:"puidsize" description:"Initial size of path unique identifiers in bytes, maximum is 10 (x1.6 for length of string representation). When the bottom pool arrives to 90%, size increases to next available value."`
	// Maximum items number in files properties cache.
	PropCacheMaxNum int `json:"prop-cache-maxnum" yaml:"prop-cache-maxnum" long:"pcmn" description:"Maximum items number in files properties cache."`
	// Maximum items number in thumbnails cache.
	ThumbCacheMaxNum int `json:"thumb-cache-maxnum" yaml:"thumb-cache-maxnum" long:"tcmn" description:"Maximum items number in thumbnails cache."`
	// Maximum items number in converted media files cache.
	MediaCacheMaxNum int `json:"media-cache-maxnum" yaml:"media-cache-maxnum" long:"mcmn" description:"Maximum items number in converted media files cache."`
	// Expiration duration to keep opened iso-disk structures in cache from last access to it.
	DiskCacheExpire time.Duration `json:"disk-cache-expire" yaml:"disk-cache-expire" long:"dce" description:"Expiration duration to keep opened iso-disk structures in cache from last access to it."`
}

// Config is common service settings.
type Config struct {
	CfgCmdLine `json:"command-line" yaml:"command-line" group:"Data Parameters"`
	CfgJWTAuth `json:"authentication" yaml:"authentication" group:"Authentication"`
	CfgWebServ `json:"webserver" yaml:"webserver" group:"Web Server"`
	CfgAppConf `json:"specification" yaml:"specification" group:"Application settings"`
}

// Instance of common service settings.
var cfg = Config{ // inits default values:
	CfgJWTAuth: CfgJWTAuth{
		AccessTTL:  time.Duration(1*24) * time.Hour,
		RefreshTTL: time.Duration(3*24) * time.Hour,
		AccessKey:  "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479",
		RefreshKey: "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie",
	},
	CfgWebServ: CfgWebServ{
		AutoCert:          false,
		PortHTTP:          []string{":80"},
		PortTLS:           []string{},
		ReadTimeout:       time.Duration(15) * time.Second,
		ReadHeaderTimeout: time.Duration(15) * time.Second,
		WriteTimeout:      time.Duration(15) * time.Second,
		IdleTimeout:       time.Duration(60) * time.Second,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   time.Duration(15) * time.Second,
	},
	CfgAppConf: CfgAppConf{
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

func init() {
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
}

const (
	gitname = "hms"
	gitpath = "github.com/schwarzlichtbezirk/" + gitname
	cfgbase = gitname
	cfgfile = "settings.yaml"

	pcfile = "pathcache.yaml"
	dcfile = "dircache.yaml"
	pffile = "profiles.yaml"
	ulfile = "userlist.yaml"
)

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
	if cfg.ConfigPath != "" {
		path = envfmt(cfg.ConfigPath)
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
		log.Printf("no access to pointed configuration path '%s'\n", cfg.ConfigPath)
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
	// try to find in config subdirectory of current path
	if ok, _ = pathexists(filepath.Join(cfgbase, cfgfile)); ok {
		retpath = cfgbase
		return
	}
	// try to find in current path
	if ok, _ = pathexists(cfgfile); ok {
		retpath = "."
		return
	}
	// check up current path is the git root path
	if ok, _ = pathexists(filepath.Join("config", cfgfile)); ok {
		retpath = "config"
		return
	}

	// check up running in devcontainer workspace
	path = filepath.Join("/workspaces", gitname, "config")
	if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
		retpath = path
		return
	}

	// check up git source path
	var prefix string
	if prefix, ok = os.LookupEnv("GOPATH"); ok {
		path = filepath.Join(prefix, "src", gitpath, "config")
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
	}

	// if GOBIN or GOPATH is present
	if prefix, ok = os.LookupEnv("GOBIN"); !ok {
		if prefix, ok = os.LookupEnv("GOPATH"); ok {
			prefix = filepath.Join(prefix, "bin")
		}
	}
	if ok {
		// try to get from go bin config
		path = filepath.Join(prefix, cfgbase)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to get from go bin root
		if ok, _ = pathexists(filepath.Join(prefix, cfgfile)); ok {
			retpath = prefix
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}

// PackPath determines package path, depended on what directory is exist.
var PackPath string

// ErrNoPack is "no package path was found" error message.
var ErrNoPack = errors.New("no package path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectPackPath() (retpath string, err error) {
	var ok bool
	var path string
	var exepath = filepath.Dir(os.Args[0])

	// try to get from environment setting
	if cfg.PackPath != "" {
		path = envfmt(cfg.PackPath)
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
		log.Printf("no access to pointed package path '%s'\n", cfg.PackPath)
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
	var prefix string
	if prefix, ok = os.LookupEnv("GOBIN"); ok {
		if ok, _ = pathexists(filepath.Join(prefix, cfg.WPKName)); ok {
			retpath = prefix
			return
		}
	}

	// if GOPATH is present
	if prefix, ok = os.LookupEnv("GOPATH"); ok {
		// try to get from go bin root
		path = filepath.Join(prefix, "bin")
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
