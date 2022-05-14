package hms

import (
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const utf8bom = "\xef\xbb\xbf"

// WriteYaml writes "data" object to YAML-file with given file name.
// File writes in UTF-8 format with BOM, and "intro" comment.
func WriteYaml(fname, intro string, data interface{}) (err error) {
	var file *os.File
	if file, err = os.OpenFile(path.Join(ConfigPath, fname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer file.Close()

	if _, err = file.WriteString(utf8bom); err != nil {
		return
	}
	if _, err = file.WriteString(intro); err != nil {
		return
	}

	var body []byte
	if body, err = yaml.Marshal(data); err != nil {
		return
	}
	if _, err = file.Write(body); err != nil {
		return
	}
	return
}

// ReadYaml reads "data" object from YAML-file
// with given file name.
func ReadYaml(fname string, data interface{}) (err error) {
	var body []byte
	if body, err = os.ReadFile(path.Join(ConfigPath, fname)); err != nil {
		return
	}
	if err = yaml.Unmarshal(body, data); err != nil {
		return
	}
	return
}

// ReadYaml reads content of PathCache structure from YAML-file
// with given file name.
func (pc *PathCache) ReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, &pc.keypath); err != nil {
		return
	}

	pc.pathkey = make(map[string]Puid_t, len(pc.keypath))
	for key, fpath := range pc.keypath {
		pc.pathkey[fpath] = key
	}
	return
}

// WriteYaml writes content of PathCache object in YAML format
// with header comment to file with given file name.
func (pc *PathCache) WriteYaml(fname string) error {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is path unique ID encoded to base32 (RFC
# 4648), values are associated paths. Those keys
# used for files paths representations in URLs. You
# can modify keys to any alphanumerical text that
# should be unique.

`
	return WriteYaml(fname, intro, pc.keypath)
}

// ReadYaml reads content of DirCache structure from YAML-file
// with given file name.
func (dc *DirCache) ReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, &dc.keydir); err != nil {
		return
	}
	return
}

// WriteYaml writes content of DirCache object in YAML format
// with header comment to file with given file name.
func (dc *DirCache) WriteYaml(fname string) error {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is path unique ID encoded to base32 (RFC
# 4648), values are associated directory properties.
# Those cache is used for directories representation
# and media groups representation. Count set format:
# [misc, video, audio, image, books, txt, arch, dir]

`
	return WriteYaml(fname, intro, dc.keydir)
}

// ReadYaml reads content of DirCache structure from YAML-file
// with given file name.
func (gc *GpsCache) ReadYaml(fname string) (n int, err error) {
	var m map[string]*GpsInfo
	if err = ReadYaml(fname, &m); err != nil {
		return
	}
	for k, v := range m {
		var puid = syspathcache.Cache(k)
		gc.Store(puid, v)
	}
	n = len(m)
	return
}

// WriteYaml writes content of GpsCache object in YAML format
// with header comment to file with given file name.
func (gc *GpsCache) WriteYaml(fname string) error {
	const intro = `
# Map with PUID/GpsInfo pairs. Contains GPS-coordinates
# and creation time from EXIF-data of scanned photos.

`
	var m = map[string]*GpsInfo{}
	gc.Range(func(key interface{}, value interface{}) bool {
		if syspath, ok := syspathcache.Path(key.(Puid_t)); ok {
			m[syspath] = value.(*GpsInfo)
		}
		return true
	})
	return WriteYaml(fname, intro, m)
}

// ReadYaml reads content of Config structure from YAML-file
// with given file name.
func (cfg *Config) ReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, &cfg); err != nil {
		return
	}
	return
}

// WriteYaml writes content of Config object in YAML format
// with header comment to file with given file name.
func (cfg *Config) WriteYaml(fname string) error {
	const intro = `
# Server configuration file. First of all you can change
# "access-key" and "refresh-key" for tokens protection.

`
	return WriteYaml(fname, intro, &cfg)
}

// ReadYaml reads content of Profiles structure from YAML-file
// with given file name.
func (pl *Profiles) ReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, &pl.list); err != nil {
		return
	}

	if len(pl.list) > 0 {
		for _, prf := range pl.list {
			Log.Infof("loaded profile id%d, login='%s'", prf.ID, prf.Login)
			// cache roots
			for _, fpath := range prf.Roots {
				syspathcache.Cache(fpath)
			}
			// cache shares
			for _, fpath := range prf.Shares {
				syspathcache.Cache(fpath)
			}

			// bring all hidden to lowercase
			for i, fpath := range prf.Hidden {
				prf.Hidden[i] = strings.ToLower(ToSlash(fpath))
			}

			// build shares tables
			prf.UpdateShares()
		}

		// check up default profile
		if prf := pl.ByID(cfg.DefAccID); prf != nil {
			if len(prf.Roots) == 0 {
				prf.FindRoots()
			}
		} else {
			Log.Fatal("default profile is not found")
		}
	} else {
		var prf = pl.NewProfile("admin", "dag qus fly in the sky")
		prf.ID = cfg.DefAccID
		Log.Infof("created profile id%d, login='%s'", prf.ID, prf.Login)
		prf.SetDefaultHidden()
		prf.FindRoots()
	}
	return
}

// WriteYaml writes content of Profiles object in YAML format
// with header comment to file with given file name.
func (pl *Profiles) WriteYaml(fname string) error {
	const intro = `
# List of administration profiles. Each profile should be with
# unique password, and allows to configure access to specified
# root drives, shares, and to hide files on specified masks.

`
	return WriteYaml(fname, intro, pl.list)
}

// ReadYaml reads content of UserCache structure from YAML-file
// with given file name.
func (uc *UserCache) ReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, &uc.list); err != nil {
		return
	}

	uc.keyuser = make(UserMap, len(uc.list))
	for _, user := range uc.list {
		user.ParseUserAgent()
		var key = UserKey(user.Addr, user.UserAgent)
		uc.keyuser[key] = user
	}
	return
}

// WriteYaml writes content of UserCache object in YAML format
// with header comment to file with given file name.
func (uc *UserCache) WriteYaml(fname string) (err error) {
	const intro = `
# Log of all clients that had activity on the server.
# Each client identify by IP-address and user-agent.

`
	return WriteYaml(fname, intro, uc.list)
}

// The End.
