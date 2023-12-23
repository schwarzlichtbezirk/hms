package config

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"

	jnt "github.com/schwarzlichtbezirk/joint"
	"github.com/schwarzlichtbezirk/wpk"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// Key to calculate user agent ID by xxhash algorithm.
	UaidHmacKey string `json:"uaid-hmac-key" yaml:"uaid-hmac-key"`
}

// CfgWebServ is web server settings.
type CfgWebServ struct {
	PortHTTP          []string      `json:"port-http" yaml:"port-http" env:"PORTHTTP" env-delim:";" short:"w" long:"http" description:"List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains."`
	PortTLS           []string      `json:"port-tls" yaml:"port-tls" env:"PORTTLS" env-delim:";" short:"s" long:"tls" description:"List of address:port values for encrypted connections. Address is skipped in most common cases, port only remains."`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout" long:"rt" description:"Maximum duration for reading the entire request, including the body."`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout" long:"rht" description:"Amount of time allowed to read request headers."`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout" long:"wt" description:"Maximum duration before timing out writes of the response."`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout" long:"it" description:"Maximum amount of time to wait for the next request when keep-alives are enabled."`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes" long:"mhb" description:"Controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line, in bytes."`
	// Maximum duration between two ajax-calls to think client is online.
	OnlineTimeout time.Duration `json:"online-timeout" yaml:"online-timeout" long:"ot" description:"Maximum duration between two ajax-calls to think client is online."`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout" long:"st" description:"Maximum duration to wait for graceful shutdown."`
}

type CfgTlsCert struct {
	// Indicates to get TLS-certificate from letsencrypt.org service if this value is true. Uses local TLS-certificate otherwise.
	UseAutoCert bool `json:"use-auto-cert" yaml:"use-auto-cert" long:"autocert" description:"Indicates to get TLS-certificate from letsencrypt.org service if this value is true. Uses local TLS-certificate otherwise."`
	// Email optionally specifies a contact email address. This is used by CAs, such as Let's Encrypt, to notify about problems with issued certificates.
	Email string `json:"email" yaml:"email" long:"email" description:"Email optionally specifies a contact email address. This is used by CAs, such as Let's Encrypt, to notify about problems with issued certificates."`
	// Creates policy where only the specified host names are allowed.
	HostWhitelist []string `json:"host-whitelist" yaml:"host-whitelist" long:"hwl" description:"Creates policy where only the specified host names are allowed."`
}

type CfgNetwork struct {
	// Timeout to establish connection to FTP-server.
	DialTimeout time.Duration `json:"dial-timeout" yaml:"dial-timeout" long:"dto" description:"Timeout to establish connection to FTP-server."`
	// Expiration duration to keep opened iso-disk structures in cache from last access to it.
	DiskCacheExpire time.Duration `json:"disk-cache-expire" yaml:"disk-cache-expire" long:"dce" description:"Expiration duration to keep opened iso-disk structures in cache from last access to it."`
}

type CfgXormDrv struct {
	// Provides XORM driver name.
	XormDriverName string `json:"xorm-driver-name" yaml:"xorm-driver-name" long:"xdn" description:"Provides XORM driver name."`
}

type CfgImgProp struct {
	// Maximum dimension of image (width x height) in megapixels to build tiles and thumbnails.
	ImageMaxMpx float32 `json:"image-max-mpx" yaml:"image-max-mpx" long:"imm" description:"Maximum dimension of image (width x height) in megapixels to build tiles and thumbnails."`
	// Stretch big image embedded into mp3-file to fit into standard icon size.
	FitEmbeddedTmb bool `json:"fit-embedded-tmb" yaml:"fit-embedded-tmb" long:"fet" description:"Stretch big image embedded into mp3-file to fit into standard icon size."`
	// Thumbnails width and height.
	TmbResolution [2]int `json:"tmb-resolution" yaml:"tmb-resolution" long:"tr" description:"Thumbnails width and height."`
	// HD images width and height.
	HDResolution [2]int `json:"hd-resolution" yaml:"hd-resolution" long:"hd" description:"HD images width and height."`
	// WebP quality of converted images from another format with original dimensions, ranges from 1 to 100 inclusive.
	MediaWebpQuality float32 `json:"media-webp-quality" yaml:"media-webp-quality" long:"mediawq" description:"WebP quality of converted images from another format with original resolution, ranges from 1 to 100 inclusive."`
	// WebP quality of converted to HD-resolution images, ranges from 1 to 100 inclusive.
	HDWebpQuality float32 `json:"hd-webp-quality" yaml:"hd-webp-quality" long:"hdwq" description:"WebP quality of converted to HD-resolution images, ranges from 1 to 100 inclusive."`
	// WebP quality of any tiles, ranges from 1 to 100 inclusive.
	TileWebpQuality float32 `json:"tile-webp-quality" yaml:"tile-webp-quality" long:"tilewq" description:"WebP quality of any tiles, ranges from 1 to 100 inclusive."`
	// WebP quality of thumbnails, ranges from 1 to 100 inclusive.
	TmbWebpQuality float32 `json:"tmb-webp-quality" yaml:"tmb-webp-quality" long:"tmbwq" description:"WebP quality of thumbnails, ranges from 1 to 100 inclusive."`
	// Number of image processing threads in which performs converting to
	// tiles and thumbnails. Zero sets this number to GOMAXPROCS value.
	ScanThreadsNum int `json:"scan-threads-num" yaml:"scan-threads-num" long:"stn" description:"Number of image processing threads in which performs converting to tiles and thumbnails."`
}

// CfgAppSets is settings for application-specific logic.
type CfgAppSets struct {
	// Name of wpk-file with program resources.
	WPKName []string `json:"wpk-name" yaml:"wpk-name,flow" long:"wpk" description:"Name of wpk-file with program resources."`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap" long:"mmap" description:"Memory mapping technology for WPK, or load into one solid byte slice otherwise."`
	// Maximum size in megabytes of embedded thumbnails memory cache.
	ThumbCacheMaxSize float32 `json:"thumb-cache-max-size" yaml:"thumb-cache-max-size" long:"pcmn" description:"Maximum size in megabytes of embedded thumbnails memory cache."`
	// Maximum size in megabytes of memory cache for converted images.
	ImgCacheMaxSize float32 `json:"img-cache-max-size" yaml:"img-cache-max-size" long:"hcmn" description:"Maximum size in megabytes of memory cache for converted images."`
	// Maximum number of photos to get on default map state.
	RangeSearchAny int `json:"range-search-any" yaml:"range-search-any" long:"rsa" description:"Maximum number of photos to get on default map state."`
	// Limit of range search.
	RangeSearchLimit int `json:"range-search-limit" yaml:"range-search-limit" long:"rsmn" description:"Limit of range search."`
}

// Config is common service settings.
type Config struct {
	CfgJWTAuth  `json:"authentication" yaml:"authentication"`
	CfgWebServ  `json:"web-server" yaml:"web-server"`
	CfgTlsCert  `json:"tls-certificates" yaml:"tls-certificates"`
	*jnt.Config `json:"network" yaml:"network"`
	CfgXormDrv  `json:"xorm" yaml:"xorm"`
	CfgImgProp  `json:"images-prop" yaml:"images-prop"`
	CfgAppSets  `json:"specification" yaml:"specification"`
}

// Instance of common service settings.
var Cfg = &Config{ // inits default values:
	CfgJWTAuth: CfgJWTAuth{
		AccessTTL:   1 * 24 * time.Hour,
		RefreshTTL:  3 * 24 * time.Hour,
		AccessKey:   "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479",
		RefreshKey:  "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie",
		UaidHmacKey: "hms-ua",
	},
	CfgWebServ: CfgWebServ{
		PortHTTP:          []string{":80"},
		PortTLS:           []string{},
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		OnlineTimeout:     3 * 60 * time.Second,
		ShutdownTimeout:   15 * time.Second,
	},
	CfgTlsCert: CfgTlsCert{
		UseAutoCert:   false,
		Email:         "example@example.org",
		HostWhitelist: []string{"example.org", "www.example.org"},
	},
	Config: &jnt.Cfg,
	CfgXormDrv: CfgXormDrv{
		XormDriverName: "sqlite3",
	},
	CfgImgProp: CfgImgProp{
		ImageMaxMpx:      46.8, // 8K photos, 8368 x 5584 (Leica Q2)
		FitEmbeddedTmb:   true,
		TmbResolution:    [2]int{256, 256},
		MediaWebpQuality: 80,
		HDWebpQuality:    75,
		TileWebpQuality:  60,
		TmbWebpQuality:   75,
		ScanThreadsNum:   4,
	},
	CfgAppSets: CfgAppSets{
		WPKName:           []string{"hms-app.wpk", "hms-edge.wpk"},
		WPKmmap:           false,
		ThumbCacheMaxSize: 64,
		ImgCacheMaxSize:   256,
		RangeSearchAny:    20,
		RangeSearchLimit:  100,
	},
}

var (
	// compiled binary version, sets by compiler with command
	//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/hms/config.BuildVers=%buildvers%'"
	BuildVers string

	// compiled binary build date, sets by compiler with command
	//    go build -ldflags="-X 'github.com/schwarzlichtbezirk/hms/config.BuildTime=%buildtime%'"
	BuildTime string
)

var (
	// Developer mode, running at debugger.
	DevMode bool
	// AppName is name of this application without extension.
	AppName = BaseName(os.Args[0])
	// Executable path.
	ExePath string
	// Configuration file with path.
	CfgFile string
	// Configuration path.
	CfgPath string
	// PkgPath determines resources packages path.
	PkgPath string
	// TmbPath determines images cache path.
	TmbPath string
)

// BaseName returns name of file in given file path without extension.
func BaseName(fpath string) string {
	var j = len(fpath)
	if j == 0 {
		return ""
	}
	var i = j - 1
	for {
		if os.IsPathSeparator(fpath[i]) {
			i++
			break
		}
		if fpath[i] == '.' {
			j = i
		}
		if i == 0 {
			break
		}
		i--
	}
	return fpath[i:j]
}

func InitConfig() {
	var err error

	if DevMode {
		Log.Info("*running in developer mode*")
	}
	Log.Infof("version: %s, builton: %s", BuildVers, BuildTime)

	if str, err := os.Executable(); err == nil {
		ExePath = path.Dir(str)
	} else {
		ExePath = path.Dir(os.Args[0])
	}

	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		const cfgsub = "config"
		// Search config in home directory with name "hms" (without extension).
		viper.SetConfigName("hms")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(path.Join(ExePath, cfgsub))
		viper.AddConfigPath(ExePath)
		viper.AddConfigPath(cfgsub)
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/" + cfgsub)
		viper.AddConfigPath("$HOME")
		viper.AddConfigPath("$GOPATH/bin/" + cfgsub)
		viper.AddConfigPath("$GOPATH/bin")
	}

	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err != nil {
		Log.Info("Config file not found!")
	} else {
		CfgFile = viper.ConfigFileUsed()
		Log.Info("Using config file:", CfgFile)
		CfgPath = filepath.Dir(CfgFile)

		if err = viper.Unmarshal(&Cfg); err != nil {
			cobra.CheckErr(err)
		}
	}
}

// CheckPath is short variant of path existence check.
func CheckPath(fpath string, fname string) (string, bool) {
	if ok, _ := wpk.PathExists(path.Join(fpath, fname)); !ok {
		return "", false
	}
	return fpath, true
}

// ConfigPath determines configuration path, depended on what directory is exist.
/*var ConfigPath string

// ErrNoCongig is "no configuration path was found" error message.
var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectConfigPath() (retpath string, err error) {
	var detectname = "hms.yaml"
	var ok bool
	var fpath string

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("CONFIGPATH"); ok {
		fpath = ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, detectname); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(ExePath, fpath), detectname); ok {
			return
		}
		Log.Warnf("no access to pointed configuration path '%s'", fpath)
	}

	// try to get from config subdirectory on executable path
	if retpath, ok = CheckPath(path.Join(ExePath, cfgdest), detectname); ok {
		return
	}
	// try to find in executable path
	if retpath, ok = CheckPath(ExePath, detectname); ok {
		return
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
		if retpath, ok = CheckPath(path.Join(fpath, cfgdest), detectname); ok {
			return
		}
		// try to get from go bin root
		if retpath, ok = CheckPath(fpath, detectname); ok {
			return
		}
	}

	// try to find in config subdirectory of current path
	if retpath, ok = CheckPath(path.Join(CurPath, cfgdest), detectname); ok {
		return
	}
	// try to find in current path
	if retpath, ok = CheckPath(CurPath, detectname); ok {
		return
	}

	// check up running from debugger
	if DevMode {
		retpath = path.Join(ExePath, "..", cfgbase)
		return
	}
	// check up running in devcontainer workspace
	if retpath, ok = CheckPath(path.Join("/workspaces", gitname, cfgbase), detectname); ok {
		return
	}

	// check up git source path
	if DevMode {
		if retpath, ok = CheckPath(path.Join(GitPath, cfgbase), detectname); ok {
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}*/

// ErrNoPack is "no package path was found" error message.
var ErrNoPack = errors.New("no package path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectPackPath() (retpath string, err error) {
	var detectname = Cfg.WPKName[0]
	var ok bool
	var fpath string

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("PACKPATH"); ok {
		fpath = filepath.ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, detectname); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(ExePath, fpath), detectname); ok {
			return
		}
		Log.Warnf("no access to pointed package path '%s'", fpath)
	}

	// try to find in executable path
	if retpath, ok = CheckPath(ExePath, detectname); ok {
		return
	}
	// try to find in current path
	if retpath, ok = CheckPath(".", detectname); ok {
		return
	}
	// try to find at parental of cofiguration path
	if retpath, ok = CheckPath(path.Join(CfgPath, ".."), detectname); ok {
		return
	}

	// if GOBIN or GOPATH is present
	if fpath, ok = os.LookupEnv("GOBIN"); !ok {
		if fpath, ok = os.LookupEnv("GOPATH"); ok {
			fpath = path.Join(fpath, "bin")
		}
	}
	if ok {
		fpath = filepath.ToSlash(fpath)
		// try to get from go bin root
		if retpath, ok = CheckPath(fpath, detectname); ok {
			return
		}
	}

	// no package was found
	err = ErrNoPack
	return
}

// DetectCachePath finds configuration path with existing configuration file at least.
func DetectCachePath() (retpath string, err error) {
	var ok bool
	var fpath string

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("CACHEPATH"); ok {
		fpath = filepath.ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, ""); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(ExePath, fpath), ""); ok {
			return
		}
		Log.Warnf("no access to pointed cache path '%s'", fpath)
	}

	retpath = path.Join(PkgPath, "cache")

	err = os.MkdirAll(retpath, os.ModePerm)
	return
}

// The End.
