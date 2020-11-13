package hms

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const utf8bom = "\xef\xbb\xbf"

func WriteYaml(fpath, intro string, data interface{}) (err error) {
	var file *os.File
	if file, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
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

func ReadYaml(fpath string, data interface{}) (err error) {
	var body []byte
	if body, err = ioutil.ReadFile(fpath); err != nil {
		return
	}
	if err = yaml.Unmarshal(body, data); err != nil {
		return
	}
	return
}

func (pc *PathCache) Load(fpath string) (err error) {
	if err = ReadYaml(fpath, &pc.keypath); err != nil {
		return
	}

	pc.pathkey = make(map[string]string, len(pc.keypath))
	for key, path := range pc.keypath {
		pc.pathkey[path] = key
	}

	// cache categories paths
	for _, path := range CatPath {
		pc.Cache(path)
	}
	return
}

func (pc *PathCache) Save(fpath string) error {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is path unique ID encoded to base32 (RFC
# 4648), values are associated paths. Those keys
# used for files paths representations in URLs. You
# can modify keys to any alphanumerical text that
# should be unique.

`
	return WriteYaml(fpath, intro, pc.keypath)
}

func (dc *DirCache) Load(fpath string) (err error) {
	if err = ReadYaml(fpath, &dc.keydir); err != nil {
		return
	}
	return
}

func (dc *DirCache) Save(fpath string) error {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is path unique ID encoded to base32 (RFC
# 4648), values are associated directory properties.
# Those cache is used for directories representation
# and media groups representation. Count set format:
# [misc, video, audio, image, books, txt, arch, dir]

`
	return WriteYaml(fpath, intro, dc.keydir)
}

func (cfg *Config) Load(fpath string) (err error) {
	if err = ReadYaml(fpath, &cfg); err != nil {
		return
	}
	return
}

func (cfg *Config) Save(fpath string) error {
	const intro = `
# Server configuration file. First of all you can change
# "access-key" and "refresh-key" for tokens protection.

`
	return WriteYaml(fpath, intro, &cfg)
}

func (al *Accounts) Load(fpath string) (err error) {
	if err = ReadYaml(fpath, &al.list); err != nil {
		return
	}

	if len(al.list) > 0 {
		for _, acc := range al.list {
			Log.Printf("loaded account id%d, login='%s'", acc.ID, acc.Login)
			// cache roots
			for _, path := range acc.Roots {
				pathcache.Cache(path)
			}
			// cache shares
			for _, path := range acc.Shares {
				pathcache.Cache(path)
			}

			// bring all hidden to lowercase
			for i, path := range acc.Hidden {
				acc.Hidden[i] = strings.ToLower(filepath.ToSlash(path))
			}

			// build shares tables
			acc.UpdateShares()
		}

		// check up default account
		if acc := al.ByID(cfg.DefAccID); acc != nil {
			if len(acc.Roots) == 0 {
				acc.FindRoots()
			}
		} else {
			Log.Fatal("default account is not found")
		}
	} else {
		var acc = al.NewAccount("admin", "dag qus fly in the sky")
		acc.ID = cfg.DefAccID
		Log.Printf("created account id%d, login='%s'", acc.ID, acc.Login)
		acc.SetDefaultHidden()
		acc.FindRoots()
	}
	return
}

func (al *Accounts) Save(fpath string) error {
	const intro = `
# List of administrators accounts. Each account should be
# with unique password, and allows to configure access to
# specified root drives, shares, and to hide files on
# specified masks.

`
	return WriteYaml(fpath, intro, al.list)
}

func (uc *UserCache) Load(fpath string) (err error) {
	if err = ReadYaml(fpath, &uc.list); err != nil {
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

func (uc *UserCache) Save(fpath string) (err error) {
	const intro = `
# Log of all clients that had activity on the server.
# Each client identify by IP-address and user-agent.

`
	return WriteYaml(fpath, intro, uc.list)
}

// The End.
