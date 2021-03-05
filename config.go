package hms

import (
	"time"
)

const (
	rootsuff = "hms"
	asstsuff = "assets"  // relative path to assets folder
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
	plugsuff = "plugin"  // relative path to third party code
	confsuff = "conf"    // relative path to configuration files folder
	tmplsuff = "tmpl"    // relative path to html templates folder
	csrcsuff = "src/github.com/schwarzlichtbezirk/hms"
)

var starttime = time.Now() // save server start time

var (
	destpath string // contains program destination path
	confpath string // configuration folder path
)

// CfgAuth is authentication settings.
type CfgAuth struct {
	// Access token time to live.
	AccessTTL int `json:"access-ttl" yaml:"access-ttl"`
	// Refresh token time to live.
	RefreshTTL int `json:"refresh-ttl" yaml:"refresh-ttl"`
	// Key for access HS-256 JWT-tokens.
	AccessKey string `json:"access-key" yaml:"access-key"`
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string `json:"refresh-key" yaml:"refresh-key"`
}

// CfgServ is web server settings.
type CfgServ struct {
	AutoCert          bool     `json:"auto-cert" yaml:"auto-cert"`
	PortHTTP          []string `json:"port-http" yaml:"port-http"`
	PortTLS           []string `json:"port-tls" yaml:"port-tls"`
	ReadTimeout       int      `json:"read-timeout" yaml:"read-timeout"`
	ReadHeaderTimeout int      `json:"read-header-timeout" yaml:"read-header-timeout"`
	WriteTimeout      int      `json:"write-timeout" yaml:"write-timeout"`
	IdleTimeout       int      `json:"idle-timeout" yaml:"idle-timeout"`
	MaxHeaderBytes    int      `json:"max-header-bytes" yaml:"max-header-bytes"`
	ShutdownTimeout   int      `json:"shutdown-timeout" yaml:"shutdown-timeout"`
}

// CfgSpec is settings for application-specific logic.
type CfgSpec struct {
	// Name of wpk-file with program resources.
	WPKName string `json:"wpk-name" yaml:"wpk-name"`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap"`
	// Maximum timeout in milliseconds between two ajax-calls to think client is online.
	OnlineTimeout int64 `json:"online-timeout" yaml:"online-timeout"`
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
		AccessTTL:  1 * 24 * 60 * 60,
		RefreshTTL: 3 * 24 * 60 * 60,
		AccessKey:  "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479",
		RefreshKey: "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie",
	},
	CfgServ: CfgServ{
		AutoCert:          false,
		PortHTTP:          []string{},
		PortTLS:           []string{},
		ReadTimeout:       15,
		ReadHeaderTimeout: 15,
		WriteTimeout:      15,
		IdleTimeout:       60,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   15,
	},
	CfgSpec: CfgSpec{
		WPKName:          "hms.wpk",
		WPKmmap:          false,
		OnlineTimeout:    3 * 60 * 1000,
		DefAccID:         1,
		ThumbFileMaxSize: 4096*3072*4 + 65536,
		PUIDsize:         3,
		PropCacheMaxNum:  32 * 1024,
		ThumbCacheMaxNum: 2 * 1024,
		MediaCacheMaxNum: 64,
	},
}

// The End.
