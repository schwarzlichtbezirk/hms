package hms

import (
	"errors"
	"os"
	"path"
	"time"
)

const (
	gitname = "hms"
	gitpath = "github.com/schwarzlichtbezirk/" + gitname
	cfgfile = "settings.yaml"

	pthfile = "pathcache.yaml"
	dirfile = "dircache.yaml"
	gpsfile = "gpscache.yaml"
	prffile = "profiles.yaml"
	usrfile = "userlist.yaml"

	tmbfile = "thumb.wpt"
	tilfile = "tiles.wpt"
)

const (
	asstsuff = "assets"  // relative path to assets folder
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
	plugsuff = "plugin"  // relative path to third party code
	confsuff = "config"  // relative path to configuration files folder
	tmplsuff = "tmpl"    // relative path to html templates folder
)

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
	PortHTTP          []string      `json:"port-http" yaml:"port-http" env:"PORTHTTP" env-delim:";" short:"w" long:"http" description:"List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains."`
	PortTLS           []string      `json:"port-tls" yaml:"port-tls" env:"PORTTLS" env-delim:";" short:"s" long:"tls" description:"List of address:port values for encrypted connections. Address is skipped in most common cases, port only remains."`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout" long:"rt" description:"Maximum duration for reading the entire request, including the body."`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout" long:"rht" description:"Amount of time allowed to read request headers."`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout" long:"wt" description:"Maximum duration before timing out writes of the response."`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout" long:"it" description:"Maximum amount of time to wait for the next request when keep-alives are enabled."`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes" long:"mhb" description:"Controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line, in bytes."`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout" long:"st" description:"Maximum duration to wait for graceful shutdown."`
}

type CfgImgProp struct {
	// Maximum size of image to make thumbnail.
	ThumbFileMaxSize int64 `json:"thumb-file-maxsize" yaml:"thumb-file-maxsize" long:"tfms" description:"Maximum size of image to make thumbnail."`
	// Use JPEG thumbnails embedded into image.
	UseEmbeddedTmb bool `json:"use-embedded-tmb" yaml:"use-embedded-tmb" long:"uet" description:"Use JPEG thumbnails embedded into image."`
	// Stretch big image embedded into mp3-file to fit into standard icon size.
	FitEmbeddedTmb bool `json:"fit-embedded-tmb" yaml:"fit-embedded-tmb" long:"fet" description:"Stretch big image embedded into mp3-file to fit into standard icon size."`
	// Thumbnails width and height.
	TmbResolution [2]int `json:"tmb-resolution" yaml:"tmb-resolution" long:"tr" description:"Thumbnails width and height."`
	// HD images width and height.
	HDResolution [2]int `json:"hd-resolution" yaml:"hd-resolution" long:"hd" description:"HD images width and height."`
	// Thumbnails JPEG quality, ranges from 1 to 100 inclusive.
	TmbJpegQuality int `json:"tmb-jpeg-quality" yaml:"tmb-jpeg-quality" long:"tjq" description:"Thumbnails JPEG quality, ranges from 1 to 100 inclusive."`
	// Number of image processing threads in which performs converting to
	// tiles and thumbnails. Zero sets this number to GOMAXPROCS value.
	ScanThreadsNum int `json:"scan-threads-num" yaml:"scan-threads-num" long:"stn" description:"Number of image processing threads in which performs converting to tiles and thumbnails."`
}

// CfgAppSets is settings for application-specific logic.
type CfgAppSets struct {
	// Name of wpk-file with program resources.
	WPKName string `json:"wpk-name" yaml:"wpk-name" long:"wpk" description:"Name of wpk-file with program resources."`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap" long:"mmap" description:"Memory mapping technology for WPK, or load into one solid byte slice otherwise."`
	// Maximum duration between two ajax-calls to think client is online.
	OnlineTimeout time.Duration `json:"online-timeout" yaml:"online-timeout" long:"ot" description:"Maximum duration between two ajax-calls to think client is online."`
	// Default profile identifier for user on localhost.
	DefAccID ID_t `json:"default-profile-id" yaml:"default-profile-id" long:"defaid" description:"Default profile identifier for user on localhost."`
	// Initial length of path unique identifiers in base32 symbols, maximum is 12
	// (x5 for length in bits).
	// When the bottom pool arrives to 90%, length increases to next available value.
	PUIDlen int `json:"puid-length" yaml:"puid-length" long:"puidlen" description:"Initial length of path unique identifiers in base32 symbols, maximum is 12 (x5 for length in bits). When the bottom pool arrives to 90%, length increases to next available value."`
	// Maximum items number in files properties cache.
	PropCacheMaxNum int `json:"prop-cache-maxnum" yaml:"prop-cache-maxnum" long:"pcmn" description:"Maximum items number in files properties cache."`
	// Maximum items number in converted media files cache.
	MediaCacheMaxNum int `json:"media-cache-maxnum" yaml:"media-cache-maxnum" long:"mcmn" description:"Maximum items number in converted media files cache."`
	// Expiration duration to keep opened iso-disk structures in cache from last access to it.
	DiskCacheExpire time.Duration `json:"disk-cache-expire" yaml:"disk-cache-expire" long:"dce" description:"Expiration duration to keep opened iso-disk structures in cache from last access to it."`
}

// Config is common service settings.
type Config struct {
	CfgJWTAuth `json:"authentication" yaml:"authentication" group:"Authentication"`
	CfgWebServ `json:"web-server" yaml:"web-server" group:"Web server"`
	CfgImgProp `json:"images-prop" yaml:"images-prop" group:"Images properties"`
	CfgAppSets `json:"specification" yaml:"specification" group:"Application settings"`
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
	CfgImgProp: CfgImgProp{
		ThumbFileMaxSize: 4096*3072*4 + 65536,
		UseEmbeddedTmb:   true,
		FitEmbeddedTmb:   true,
		TmbResolution:    [2]int{256, 256},
		TmbJpegQuality:   80,
		ScanThreadsNum:   2,
	},
	CfgAppSets: CfgAppSets{
		WPKName:          "hms-full.wpk",
		WPKmmap:          false,
		OnlineTimeout:    time.Duration(3*60*1000) * time.Millisecond,
		DefAccID:         1,
		PUIDlen:          5,
		PropCacheMaxNum:  32 * 1024,
		MediaCacheMaxNum: 64,
		DiskCacheExpire:  time.Duration(15) * time.Second,
	},
}

// compiled binary version, sets by compiler with command
//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/hms.buildvers=%buildvers%'"
var buildvers string

// compiled binary build date, sets by compiler with command
//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/hms.builddate=%date%'"
var builddate string

// compiled binary build time, sets by compiler with command
//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/hms.buildtime=%time%'"
var buildtime string

// save server start time
var starttime = time.Now()

const cfgbase = "config"

// ConfigPath determines configuration path, depended on what directory is exist.
var ConfigPath string

// ErrNoCongig is "no configuration path was found" error message.
var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectConfigPath() (retpath string, err error) {
	var ok bool
	var fpath string
	var exepath = path.Dir(ToSlash(os.Args[0]))

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("CONFIGPATH"); ok {
		fpath = ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, cfgfile); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(exepath, fpath), cfgfile); ok {
			return
		}
		Log.Infof("no access to pointed configuration path '%s'\n", fpath)
	}

	// try to get from config subdirectory on executable path
	if retpath, ok = CheckPath(path.Join(exepath, cfgbase), cfgfile); ok {
		return
	}
	// try to find in executable path
	if retpath, ok = CheckPath(exepath, cfgfile); ok {
		return
	}
	// try to find in config subdirectory of current path
	if retpath, ok = CheckPath(cfgbase, cfgfile); ok {
		return
	}
	// try to find in current path
	if retpath, ok = CheckPath(".", cfgfile); ok {
		return
	}
	// check up current path is the git root path
	if retpath, ok = CheckPath(cfgbase, cfgfile); ok {
		return
	}

	// check up running in devcontainer workspace
	if retpath, ok = CheckPath(path.Join("/workspaces", gitname, cfgbase), cfgfile); ok {
		return
	}

	// check up git source path
	if fpath, ok = os.LookupEnv("GOPATH"); ok {
		if retpath, ok = CheckPath(path.Join(ToSlash(fpath), "src", gitpath, cfgbase), cfgfile); ok {
			return
		}
	}

	// if GOBIN or GOPATH is present
	if fpath, ok = os.LookupEnv("GOBIN"); !ok {
		if fpath, ok = os.LookupEnv("GOPATH"); ok {
			fpath = path.Join(fpath, "bin")
		}
	}
	if ok {
		fpath = ToSlash(fpath)
		// try to get from go bin config
		if retpath, ok = CheckPath(path.Join(fpath, cfgbase), cfgfile); ok {
			return
		}
		// try to get from go bin root
		if retpath, ok = CheckPath(fpath, cfgfile); ok {
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}

// PackPath determines resources package path, depended on what directory is exist.
var PackPath string

// ErrNoPack is "no package path was found" error message.
var ErrNoPack = errors.New("no package path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectPackPath() (retpath string, err error) {
	var ok bool
	var fpath string
	var exepath = path.Dir(ToSlash(os.Args[0]))

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("PACKPATH"); ok {
		fpath = ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, cfg.WPKName); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(exepath, fpath), cfg.WPKName); ok {
			return
		}
		Log.Infof("no access to pointed package path '%s'\n", fpath)
	}

	// try to find in executable path
	if retpath, ok = CheckPath(exepath, cfg.WPKName); ok {
		return
	}
	// try to find in current path
	if retpath, ok = CheckPath(".", cfg.WPKName); ok {
		return
	}
	// try to find at parental of cofiguration path
	if retpath, ok = CheckPath(path.Join(ConfigPath, ".."), cfg.WPKName); ok {
		return
	}

	// if GOBIN is present
	if fpath, ok = os.LookupEnv("GOBIN"); ok {
		if retpath, ok = CheckPath(ToSlash(fpath), cfg.WPKName); ok {
			return
		}
	}

	// if GOPATH is present
	if fpath, ok = os.LookupEnv("GOPATH"); ok {
		// try to get from go bin root
		if retpath, ok = CheckPath(path.Join(ToSlash(fpath), "bin"), cfg.WPKName); ok {
			return
		}
	}

	// no package was found
	err = ErrNoPack
	return
}

// CachePath determines images cache path, depended on what directory is exist.
var CachePath string

// DetectCachePath finds configuration path with existing configuration file at least.
func DetectCachePath() (retpath string, err error) {
	var ok bool
	var fpath string
	var exepath = path.Dir(ToSlash(os.Args[0]))

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("CACHEPATH"); ok {
		fpath = ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, ""); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(exepath, fpath), ""); ok {
			return
		}
		Log.Infof("no access to pointed cache path '%s'\n", fpath)
	}

	retpath = path.Join(PackPath, "cache")
	return
}

// The End.
