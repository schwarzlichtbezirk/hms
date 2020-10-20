package hms

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const utf8bom = "\xef\xbb\xbf"

func (c *KeyThumbCache) Load(fpath string) (err error) {
	var body []byte
	if body, err = ioutil.ReadFile(fpath); err == nil {
		if err = yaml.Unmarshal(body, &c.keypath); err != nil {
			return
		}
	} else {
		return
	}

	c.pathkey = make(map[string]string, len(c.keypath))
	for key, path := range c.keypath {
		c.pathkey[path] = key
	}
	return
}

func (c *KeyThumbCache) Save(fpath string) (err error) {
	const intro = `
# Here is rewritable cache with key/path pairs list.
# It's loads on server start, and saves before exit.
# Each key is MD5-hash of file system path encoded
# to base32, values are associated paths. Those keys
# used for files paths representations in URLs. You
# can modify keys to any alphanumerical text that
# should be unique.

`

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
	if body, err = yaml.Marshal(c.keypath); err != nil {
		return
	}
	if _, err = file.Write(body); err != nil {
		return
	}
	return
}

func (cfg *Config) Load(fpath string) (err error) {
	var body []byte
	if body, err = ioutil.ReadFile(fpath); err == nil {
		if err = yaml.Unmarshal(body, &cfg); err != nil {
			return
		}
	} else {
		return
	}
	return
}

func (cfg *Config) Save(fpath string) (err error) {
	const intro = `
# Server configuration file. First of all you can change
# "access-key" and "refresh-key" for tokens defence, and
# "path-hash-salt" to prevent decode file paths hashes.

`

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
	if body, err = yaml.Marshal(&cfg); err != nil {
		return
	}
	if _, err = file.Write(body); err != nil {
		return
	}
	return
}

func (al *Accounts) Load(fpath string) (err error) {
	var body []byte
	if body, err = ioutil.ReadFile(fpath); err == nil {
		if err = yaml.Unmarshal(body, &al.list); err != nil {
			return
		}
	} else {
		return
	}

	if len(al.list) > 0 {
		for _, acc := range al.list {
			Log.Printf("loaded account id%d, login='%s'", acc.ID, acc.Login)
			// bring all roots to valid slashes
			for i, path := range acc.Roots {
				acc.Roots[i] = filepath.ToSlash(path)
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

func (al *Accounts) Save(fpath string) (err error) {
	const intro = `
# List of administrators accounts. Each account should be
# with unique password, and allows to configure access to
# specified root drives, shares, and to hide files on
# specified masks.

`

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
	if body, err = yaml.Marshal(al.list); err != nil {
		return
	}
	if _, err = file.Write(body); err != nil {
		return
	}
	return
}

// The End.
