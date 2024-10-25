package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Developer mode, running at debugger.
	DevMode bool
	// AppName is name of this application without extension.
	AppName = PathName(os.Args[0])
	// Executable path.
	ExePath string
	// Configuration file with path.
	CfgFile string
	// Configuration path.
	CfgPath string
	// SQLite-files path.
	SqlPath string
	// TmbPath determines images cache path.
	TmbPath string
	// PkgPath determines resources packages path.
	PkgPath string
)

var (
	ErrNoCfgFile = errors.New("configyration file was not found")
	ErrNoPkgPath = errors.New("no packages path was found")
)

func InitConfig() {
	var err error

	if DevMode {
		Log.Info("*running in developer mode*")
	}
	Log.Infof("version: %s, builton: %s", BuildVers, BuildTime)

	ExePath = func() string {
		if str, err := os.Executable(); err == nil {
			return filepath.Dir(str)
		} else {
			return filepath.Dir(os.Args[0])
		}
	}()

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

	// Detect SQLite path.
	if SqlPath == "" {
		SqlPath = LookupInLocations("SQLPATH", "sqlite", "slot-club.sqlite")
	}
	cobra.CheckErr(os.MkdirAll(SqlPath, os.ModePerm))
	Log.Infof("sqlite path: %s", SqlPath)

	// Detect images caches path.
	if TmbPath == "" {
		TmbPath = LookupInLocations("TMBPATH", "thumbs", "thumb.wpt")
	}
	cobra.CheckErr(os.MkdirAll(TmbPath, os.ModePerm))
	Log.Infof("thumbs path: %s", TmbPath)

	// Detect packages path.
	if PkgPath == "" {
		PkgPath = LookupInLocations("PKGPATH", ".", Cfg.WPKName[0])
	}
	if ok, _ := DirExists(PkgPath); !ok {
		cobra.CheckErr(ErrNoPkgPath)
	}
	Log.Infof("package path: %s", PkgPath)
}

// PathName returns name of file in given file path without extension.
func PathName(fpath string) string {
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
func DirExists(fpath string) (bool, error) {
	var stat, err = os.Stat(fpath)
	if err == nil {
		return stat.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// FileExists check up file existence.
func FileExists(fpath string) (bool, error) {
	var stat, err = os.Stat(fpath)
	if err == nil {
		return !stat.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func LookupInLocations(env, sub, fname string) (fpath string) {
	var list []string
	if val, ok := os.LookupEnv(env); ok {
		list, _ = AddDir(list, val)
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
		list, _ = AddDir(list, filepath.Join(env, "bin", sub))
		list, _ = AddDir(list, filepath.Join(env, "bin"))
	}
	if fpath = LookupDir(list, fname); fpath == "" {
		fpath = filepath.Join(ExePath, sub)
	}
	return
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
