package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	jnt "github.com/schwarzlichtbezirk/joint"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

var (
	ErrNoCfgFile = errors.New("configyration file was not found")
	ErrNoPkgPath = errors.New("no packages path was found")
	ErrNoTmbPath = errors.New("no images caches path was found")
)

// CfgJWTAuth is authentication settings.
type CfgJWTAuth struct {
	// Access token time to live.
	AccessTTL time.Duration `json:"access-ttl" yaml:"access-ttl" mapstructure:"access-ttl"`
	// Refresh token time to live.
	RefreshTTL time.Duration `json:"refresh-ttl" yaml:"refresh-ttl" mapstructure:"refresh-ttl"`
	// Key for access HS-256 JWT-tokens.
	AccessKey string `json:"access-key" yaml:"access-key" mapstructure:"access-key"`
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string `json:"refresh-key" yaml:"refresh-key" mapstructure:"refresh-key"`
	// Key to calculate user agent ID by xxhash algorithm.
	UaidHmacKey string `json:"uaid-hmac-key" yaml:"uaid-hmac-key" mapstructure:"uaid-hmac-key"`
}

// CfgWebServ is web server settings.
type CfgWebServ struct {
	// List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains.
	PortHTTP []string `json:"port-http" yaml:"port-http" mapstructure:"port-http"`
	// List of address:port values for encrypted connections. Address is skipped in most common cases, port only remains.
	PortTLS []string `json:"port-tls" yaml:"port-tls" mapstructure:"port-tls"`
	// Maximum duration for reading the entire request, including the body.
	ReadTimeout time.Duration `json:"read-timeout" yaml:"read-timeout" mapstructure:"read-timeout"`
	// Amount of time allowed to read request headers.
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout" mapstructure:"read-header-timeout"`
	// Maximum duration before timing out writes of the response.
	WriteTimeout time.Duration `json:"write-timeout" yaml:"write-timeout" mapstructure:"write-timeout"`
	// Maximum amount of time to wait for the next request when keep-alives are enabled.
	IdleTimeout time.Duration `json:"idle-timeout" yaml:"idle-timeout" mapstructure:"idle-timeout"`
	// Controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line, in bytes.
	MaxHeaderBytes int `json:"max-header-bytes" yaml:"max-header-bytes" mapstructure:"max-header-bytes"`
	// Maximum duration between two ajax-calls to think client is online.
	OnlineTimeout time.Duration `json:"online-timeout" yaml:"online-timeout" mapstructure:"online-timeout"`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout" mapstructure:"shutdown-timeout"`
}

type CfgTlsCert struct {
	// Indicates to get TLS-certificate from letsencrypt.org service if this value is true. Uses local TLS-certificate otherwise.
	UseAutoCert bool `json:"use-auto-cert" yaml:"use-auto-cert" mapstructure:"use-auto-cert"`
	// Email optionally specifies a contact email address. This is used by CAs, such as Let's Encrypt, to notify about problems with issued certificates.
	Email string `json:"email" yaml:"email" mapstructure:"email"`
	// Creates policy where only the specified host names are allowed.
	HostWhitelist []string `json:"host-whitelist" yaml:"host-whitelist" mapstructure:"host-whitelist"`
}

type CfgNetwork struct {
	// Timeout to establish connection to FTP-server.
	DialTimeout time.Duration `json:"dial-timeout" yaml:"dial-timeout" mapstructure:"dial-timeout"`
	// Expiration duration to keep opened iso-disk structures in cache from last access to it.
	DiskCacheExpire time.Duration `json:"disk-cache-expire" yaml:"disk-cache-expire" mapstructure:"disk-cache-expire"`
}

type CfgXormDrv struct {
	// Provides XORM driver name.
	XormDriverName string `json:"xorm-driver-name" yaml:"xorm-driver-name" mapstructure:"xorm-driver-name"`
}

type CfgImgProp struct {
	// Maximum dimension of image (width x height) in megapixels to build tiles and thumbnails.
	ImageMaxMpx float32 `json:"image-max-mpx" yaml:"image-max-mpx" mapstructure:"image-max-mpx"`
	// Stretch big image embedded into mp3-file to fit into standard icon size.
	FitEmbeddedTmb bool `json:"fit-embedded-tmb" yaml:"fit-embedded-tmb" mapstructure:"fit-embedded-tmb"`
	// Thumbnails width and height.
	TmbResolution [2]int `json:"tmb-resolution" yaml:"tmb-resolution" mapstructure:"tmb-resolution"`
	// HD images width and height.
	HDResolution [2]int `json:"hd-resolution" yaml:"hd-resolution" mapstructure:"hd-resolution"`
	// WebP quality of converted images from another format with original dimensions, ranges from 1 to 100 inclusive.
	MediaWebpQuality float32 `json:"media-webp-quality" yaml:"media-webp-quality" mapstructure:"media-webp-quality"`
	// WebP quality of converted to HD-resolution images, ranges from 1 to 100 inclusive.
	HDWebpQuality float32 `json:"hd-webp-quality" yaml:"hd-webp-quality" mapstructure:"hd-webp-quality"`
	// WebP quality of any tiles, ranges from 1 to 100 inclusive.
	TileWebpQuality float32 `json:"tile-webp-quality" yaml:"tile-webp-quality" mapstructure:"tile-webp-quality"`
	// WebP quality of thumbnails, ranges from 1 to 100 inclusive.
	TmbWebpQuality float32 `json:"tmb-webp-quality" yaml:"tmb-webp-quality" mapstructure:"tmb-webp-quality"`
	// Number of image processing threads in which performs converting to
	// tiles and thumbnails. Zero sets this number to GOMAXPROCS value.
	ScanThreadsNum int `json:"scan-threads-num" yaml:"scan-threads-num" mapstructure:"scan-threads-num"`
}

// CfgAppSets is settings for application-specific logic.
type CfgAppSets struct {
	// Name of wpk-file with program resources.
	WPKName []string `json:"wpk-name" yaml:"wpk-name,flow" mapstructure:"wpk-name"`
	// Memory mapping technology for WPK, or load into one solid byte slice otherwise.
	WPKmmap bool `json:"wpk-mmap" yaml:"wpk-mmap" mapstructure:"wpk-mmap"`
	// Maximum size in megabytes of embedded thumbnails memory cache.
	ThumbCacheMaxSize float32 `json:"thumb-cache-max-size" yaml:"thumb-cache-max-size" mapstructure:"thumb-cache-max-size"`
	// Maximum size in megabytes of memory cache for converted images.
	ImgCacheMaxSize float32 `json:"img-cache-max-size" yaml:"img-cache-max-size" mapstructure:"img-cache-max-size"`
	// Maximum number of photos to get on default map state.
	RangeSearchAny int `json:"range-search-any" yaml:"range-search-any" mapstructure:"range-search-any"`
	// Limit of range search.
	RangeSearchLimit int `json:"range-search-limit" yaml:"range-search-limit" mapstructure:"range-search-limit"`
}

// Config is common service settings.
type Config struct {
	CfgJWTAuth  `json:"authentication" yaml:"authentication" mapstructure:"authentication"`
	CfgWebServ  `json:"web-server" yaml:"web-server" mapstructure:"web-server"`
	CfgTlsCert  `json:"tls-certificates" yaml:"tls-certificates" mapstructure:"tls-certificates"`
	*jnt.Config `json:"network" yaml:"network" mapstructure:"network"`
	CfgXormDrv  `json:"xorm" yaml:"xorm" mapstructure:"xorm"`
	CfgImgProp  `json:"images-prop" yaml:"images-prop" mapstructure:"images-prop"`
	CfgAppSets  `json:"specification" yaml:"specification" mapstructure:"specification"`
}

// Instance of common service settings.
// Inits default values if config is not found.
var Cfg = &Config{
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

func InitConfig() {
	var err error

	if DevMode {
		Log.Info("*running in developer mode*")
	}
	Log.Infof("version: %s, builton: %s", BuildVers, BuildTime)

	if str, err := os.Executable(); err == nil {
		ExePath = filepath.Dir(str)
	} else {
		ExePath = filepath.Dir(os.Args[0])
	}

	if CfgFile != "" {
		if ok, _ := FileExists(CfgFile); !ok {
			cobra.CheckErr(ErrNoCfgFile)
		}
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		const sub = "config"
		// Search config in home directory with name "hms" (without extension).
		viper.SetConfigName("hms")
		viper.SetConfigType("yaml")
		if env, ok := os.LookupEnv("CFGFILE"); ok {
			viper.AddConfigPath(env)
		}
		viper.AddConfigPath(filepath.Join(ExePath, sub))
		viper.AddConfigPath(ExePath)
		viper.AddConfigPath(sub)
		viper.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, sub))
			viper.AddConfigPath(home)
		}
		if env, ok := os.LookupEnv("GOBIN"); ok {
			viper.AddConfigPath(filepath.Join(env, sub))
			viper.AddConfigPath(env)
		} else if env, ok := os.LookupEnv("GOPATH"); ok {
			viper.AddConfigPath(filepath.Join(env, "bin", sub))
			viper.AddConfigPath(filepath.Join(env, "bin"))
		}
	}

	viper.AutomaticEnv()

	// Read config and detect config path.
	if err = viper.ReadInConfig(); err != nil {
		Log.Warn("config file not found!")
	} else {
		cobra.CheckErr(viper.Unmarshal(&Cfg))
		CfgFile = viper.ConfigFileUsed()
		CfgPath = filepath.Dir(CfgFile)
		Log.Infof("config path: %s", CfgPath)
	}

	// Detect packages path.
	if PkgPath != "" {
		if ok, _ := DirExists(PkgPath); !ok {
			cobra.CheckErr(ErrNoPkgPath)
		}
	} else {
		var list []string
		if env, ok := os.LookupEnv("PKGPATH"); ok {
			list, _ = AddDir(list, env)
		}
		list, _ = AddDir(list,
			ExePath,
			filepath.Join(CfgPath, ".."),
			CfgPath,
			".",
		)
		if home, err := os.UserHomeDir(); err == nil {
			list, _ = AddDir(list, home)
		}
		if env, ok := os.LookupEnv("GOBIN"); ok {
			list, _ = AddDir(list, env)
		} else if env, ok := os.LookupEnv("GOPATH"); ok {
			list, _ = AddDir(list, env)
		}
		PkgPath = LookupDir(list, Cfg.WPKName[0])
	}
	if PkgPath == "" {
		cobra.CheckErr(ErrNoPkgPath)
	}
	Log.Infof("package path: %s", PkgPath)

	// Detect images caches path.
	if TmbPath != "" {
		if ok, _ := DirExists(TmbPath); !ok {
			cobra.CheckErr(ErrNoTmbPath)
		}
	} else {
		const sub = "cache"
		var list []string
		if env, ok := os.LookupEnv("TMBPATH"); ok {
			list, _ = AddDir(list, env)
		}
		list, _ = AddDir(list,
			filepath.Join(ExePath, sub),
			ExePath,
			filepath.Join(CfgPath, "..", sub),
			filepath.Join(CfgPath, ".."),
			CfgPath,
			sub,
			".",
		)
		if home, err := os.UserHomeDir(); err == nil {
			list, _ = AddDir(list, filepath.Join(home, sub))
			list, _ = AddDir(list, home)
		}
		if env, ok := os.LookupEnv("GOBIN"); ok {
			list, _ = AddDir(list, filepath.Join(env, sub))
			list, _ = AddDir(list, env)
		} else if env, ok := os.LookupEnv("GOPATH"); ok {
			list, _ = AddDir(list, filepath.Join(env, sub))
			list, _ = AddDir(list, env)
		}
		TmbPath = LookupDir(list, "thumb.wpt")
	}
	if TmbPath == "" {
		TmbPath = filepath.Join(PkgPath, "cache")
		cobra.CheckErr(os.MkdirAll(TmbPath, os.ModePerm))
	}
	Log.Infof("cache path: %s", TmbPath)
}

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

// DirExists check up directory existence.
func DirExists(path string) (bool, error) {
	var stat, err = os.Stat(path)
	if err == nil {
		return stat.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// FileExists check up file existence.
func FileExists(path string) (bool, error) {
	var stat, err = os.Stat(path)
	if err == nil {
		return !stat.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func LookupDir(list []string, fname string) string {
	for _, fpath := range list {
		if ok, _ := FileExists(filepath.Join(fpath, fname)); ok {
			return fpath
		}
	}
	return ""
}

func AbsDir(dir string) (string, error) {
	dir = os.ExpandEnv(dir)
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir), nil
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return dir, err
	}
	return filepath.Clean(dir), nil
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func AddDir(list []string, dirs ...string) ([]string, error) {
	var errs []error
	var err error
	for _, dir := range dirs {
		if dir, err = AbsDir(dir); err != nil {
			errs = append(errs, err)
			continue
		}
		if StringInSlice(dir, list) {
			continue
		}
		list = append(list, dir)
	}
	return list, errors.Join(errs...)
}

// The End.
