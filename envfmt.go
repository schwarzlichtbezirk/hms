package hms

import (
	"os"
	"path/filepath"
	"regexp"
)

var (
	evure = regexp.MustCompile(`\$\{\w+\}`) // env var with unix-like syntax
	evwre = regexp.MustCompile(`\%\w+\%`)   // env var with windows-like syntax
)

func envfmt(p string) string {
	return filepath.ToSlash(evwre.ReplaceAllStringFunc(evure.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip ${...} and replace by env value
	}), func(name string) string {
		return os.Getenv(name[1 : len(name)-1]) // strip %...% and replace by env value
	}))
}

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
