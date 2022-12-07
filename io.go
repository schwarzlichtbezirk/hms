package hms

import (
	"io"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
	"xorm.io/xorm"
)

// Log is global static ring logger object.
var Log = NewLogger(os.Stderr, LstdFlags, 300)

const utf8bom = "\xef\xbb\xbf"

// WriteYaml writes "data" object to YAML-file with given file name.
// File writes in UTF-8 format with BOM, and "intro" comment.
func WriteYaml(fname, intro string, data interface{}) (err error) {
	var w io.WriteCloser
	if w, err = os.OpenFile(path.Join(ConfigPath, fname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer w.Close()

	if _, err = w.Write(s2b(utf8bom)); err != nil {
		return
	}
	if _, err = w.Write(s2b(intro)); err != nil {
		return
	}

	var body []byte
	if body, err = yaml.Marshal(data); err != nil {
		return
	}
	if _, err = w.Write(body); err != nil {
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

// YamlReadWriter allows to get common access to all structures with
// reading/writing itself to YAML-file.
type YamlReadWriter interface {
	ReadYaml(string) error
	WriteYaml(string) error
}

// ReadYaml reads content of GpsCache structure from YAML-file
// with given file name.
func (gc *GpsCache) ReadYaml(fname string) (err error) {
	var r io.ReadCloser
	if r, err = os.Open(path.Join(ConfigPath, fname)); err != nil {
		return
	}
	defer r.Close()

	type item struct {
		Path     string `json:"path" yaml:"path" xml:"path"`
		*GpsInfo `yaml:",inline"`
	}
	var dec = yaml.NewDecoder(r)
	for {
		var item item
		if err = dec.Decode(&item); err != nil {
			if err == io.EOF {
				if item.Path == "" {
					err = nil
					break
				}
			} else {
				return
			}
		}
		var puid, _ = PathStorePUID(item.Path)
		gc.Store(puid, item.GpsInfo)
	}
	return
}

// WriteYaml writes content of GpsCache object in YAML format
// with header comment to file with given file name.
func (gc *GpsCache) WriteYaml(fname string) (err error) {
	const intro = `
# Contains GPS-coordinates and creation
# time from EXIF-data of scanned photos.

`
	var w io.WriteCloser
	if w, err = os.OpenFile(path.Join(ConfigPath, fname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return
	}
	defer w.Close()

	if _, err = w.Write(s2b(utf8bom)); err != nil {
		return
	}
	if _, err = w.Write(s2b(intro)); err != nil {
		return
	}

	type item struct {
		Path     string `json:"path" yaml:"path" xml:"path"`
		*GpsInfo `yaml:",inline"`
	}
	var enc = yaml.NewEncoder(w)
	gc.Range(func(puid Puid_t, gps *GpsInfo) bool {
		if syspath, ok := PathStorePath(puid); ok {
			if err = enc.Encode(&item{syspath, gps}); err != nil {
				return false
			}
		}
		return true
	})
	enc.Close()
	return
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
		Session(xormEngine, func(session *xorm.Session) (res interface{}, err error) {
			for _, prf := range pl.list {
				Log.Infof("loaded profile id%d, login='%s'", prf.ID, prf.Login)
				// cache roots
				for _, fpath := range prf.Roots {
					PathStoreCache(fpath)
				}
				// cache shares
				for _, fpath := range prf.Shares {
					PathStoreCache(fpath)
				}

				// bring all hidden to lowercase
				for i, fpath := range prf.Hidden {
					prf.Hidden[i] = strings.ToLower(ToSlash(fpath))
				}

				// build shares tables
				prf.UpdateShares()
			}
			return
		})

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
