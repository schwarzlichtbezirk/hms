package hms

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const utf8bom = "\xef\xbb\xbf"

// WriteYaml writes "data" object to YAML-file with given file name.
// File writes in UTF-8 format with BOM, and "intro" comment.
func WriteYaml(fname, intro string, data interface{}) (err error) {
	var file *os.File
	if file, err = os.OpenFile(filepath.Join(ConfigPath, fname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
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

// ReadYaml reads "data" object from YAML-file with given file name.
func ReadYaml(fname string, data interface{}) (err error) {
	var body []byte
	if body, err = os.ReadFile(filepath.Join(ConfigPath, fname)); err != nil {
		return
	}
	if err = yaml.Unmarshal(body, data); err != nil {
		return
	}
	return
}

// Load content of PathCache structure from YAML-file with given file name.
func (pc *PathCache) Load(fname string) (err error) {
	if err = ReadYaml(fname, &pc.keypath); err != nil {
		return
	}

	pc.pathkey = make(map[string]string, len(pc.keypath))
	for key, fpath := range pc.keypath {
		pc.pathkey[fpath] = key
	}

	// cache categories paths
	for _, fpath := range CatPath {
		pc.Cache(fpath)
	}
	return
}

// Save content of PathCache object in YAML format with
// header comment to file with given file name.
func (pc *PathCache) Save(fname string) error {
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

// Load content of DirCache structure from YAML-file with given file name.
func (dc *DirCache) Load(fname string) (err error) {
	if err = ReadYaml(fname, &dc.keydir); err != nil {
		return
	}
	return
}

// Save content of DirCache object in YAML format with
// header comment to file with given file name.
func (dc *DirCache) Save(fname string) error {
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

// Load content of Config structure from YAML-file with given file name.
func (cfg *Config) Load(fname string) (err error) {
	if err = ReadYaml(fname, &cfg); err != nil {
		return
	}
	return
}

// Save content of Config object in YAML format with
// header comment to file with given file name.
func (cfg *Config) Save(fname string) error {
	const intro = `
# Server configuration file. First of all you can change
# "access-key" and "refresh-key" for tokens protection.

`
	return WriteYaml(fname, intro, &cfg)
}

// Load content of Profiles structure from YAML-file with given file name.
func (pl *Profiles) Load(fname string) (err error) {
	if err = ReadYaml(fname, &pl.list); err != nil {
		return
	}

	if len(pl.list) > 0 {
		for _, prf := range pl.list {
			Log.Printf("loaded profile id%d, login='%s'", prf.ID, prf.Login)
			// cache roots
			for _, fpath := range prf.Roots {
				pathcache.Cache(fpath)
			}
			// cache shares
			for _, fpath := range prf.Shares {
				pathcache.Cache(fpath)
			}

			// bring all hidden to lowercase
			for i, fpath := range prf.Hidden {
				prf.Hidden[i] = ToSlash(fpath)
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
		Log.Printf("created profile id%d, login='%s'", prf.ID, prf.Login)
		prf.SetDefaultHidden()
		prf.FindRoots()
	}
	return
}

// Save content of Profiles object in YAML format with
// header comment to file with given file name.
func (pl *Profiles) Save(fname string) error {
	const intro = `
# List of administration profiles. Each profile should be with
# unique password, and allows to configure access to specified
# root drives, shares, and to hide files on specified masks.

`
	return WriteYaml(fname, intro, pl.list)
}

// Load content of UserCache structure from YAML-file with given file name.
func (uc *UserCache) Load(fname string) (err error) {
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

// Save content of UserCache object in YAML format with
// header comment to file with given file name.
func (uc *UserCache) Save(fname string) (err error) {
	const intro = `
# Log of all clients that had activity on the server.
# Each client identify by IP-address and user-agent.

`
	return WriteYaml(fname, intro, uc.list)
}

// The End.
