package hms

import (
	"io"
	"net"
	"os"
	"sort"

	cfg "github.com/schwarzlichtbezirk/hms/config"

	"gopkg.in/yaml.v3"
)

const utf8bom = "\xef\xbb\xbf"

// WriteYaml writes "data" object to YAML-file with given file name.
// File writes in UTF-8 format with BOM, and "intro" comment.
func WriteYaml(fname, intro string, data any) (err error) {
	var w io.WriteCloser
	if w, err = os.OpenFile(JoinFast(cfg.ConfigPath, fname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
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
func ReadYaml(fname string, data any) (err error) {
	var body []byte
	if body, err = os.ReadFile(JoinFast(cfg.ConfigPath, fname)); err != nil {
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

// CfgReadYaml reads content of Config structure from YAML-file
// with given file name.
func CfgReadYaml(fname string) (err error) {
	if err = ReadYaml(fname, Cfg); err != nil {
		return
	}
	return
}

// CfgWriteYaml writes content of Config object in YAML format
// with header comment to file with given file name.
func CfgWriteYaml(fname string) error {
	const intro = `
# Server configuration file. First of all you can change
# "access-key" and "refresh-key" for tokens protection.

`
	return WriteYaml(fname, intro, Cfg)
}

// PrfReadYaml reads content of Profiles structure from YAML-file
// with given file name.
func PrfReadYaml(fname string) (err error) {
	var list []*Profile
	if err = ReadYaml(fname, &list); err != nil {
		return
	}
	PrfList = map[ID_t]*Profile{}
	for _, prf := range list {
		PrfList[prf.ID] = prf
	}

	if len(list) > 0 {
		SqlSession(func(session *Session) (res any, err error) {
			for _, prf := range list {
				Log.Infof("loaded profile id%d, login='%s'", prf.ID, prf.Login)
				// cache local and remote roots
				for _, dp := range prf.Roots {
					PathStoreCache(session, dp.Path)
				}
				for _, dp := range prf.Remote {
					PathStoreCache(session, dp.Path)
				}
				// cache shares
				for _, dp := range prf.Shares {
					PathStoreCache(session, dp.Path)
				}

				// bring all hidden to lowercase
				for i, fpath := range prf.Hidden {
					prf.Hidden[i] = ToLower(ToSlash(fpath))
				}

				// build shares tables
				prf.updateGrp()
				// check up some roots already defined
				if len(prf.Roots) == 0 {
					prf.FindLocal()
				}

				// print shares list for each
				for _, dp := range prf.Shares {
					if puid, ok := PathCache.GetRev(dp.Path); ok {
						Log.Infof("id%d: shared '%s' as %s", prf.ID, dp.Name, puid)
					} else {
						Log.Warnf("id%d: can not share '%s'", prf.ID, dp.Name)
					}
				}
			}
			return
		})
	} else {
		var prf = NewProfile("admin", "dag qus fly in the sky")
		prf.ID = 1
		// set hidden files array to default predefined list
		prf.Hidden = append([]string{}, DefHidden...)
		// set default "home" share
		prf.Shares = []DiskPath{
			{CPhome, CatNames[CPhome]},
		}
		// build shares tables
		prf.updateGrp()
		// setup all available disks as the roots
		prf.FindLocal()
		Log.Infof("created profile id%d, login='%s'", prf.ID, prf.Login)
	}
	return
}

// PrfWriteYaml writes content of Profiles object in YAML format
// with header comment to file with given file name.
func PrfWriteYaml(fname string) error {
	const intro = `
# List of administration profiles. Each profile should be with
# unique password, and allows to configure access to specified
# root paths, shares, and to hide files on specified masks.

`
	var list []*Profile
	for _, prf := range PrfList {
		list = append(list, prf)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return WriteYaml(fname, intro, list)
}

// ReadPasslist reads content of white list from YAML-file
// with given file name.
func ReadPasslist(fname string) (err error) {
	var list []string
	if err = ReadYaml(fname, &list); err != nil {
		return
	}
	for _, str := range list {
		if _, ipn, err := net.ParseCIDR(str); err == nil {
			Passlist = append(Passlist, *ipn)
			continue
		}
		if ip := net.ParseIP(str); ip != nil {
			Passlist = append(Passlist, net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(len(ip)*8, len(ip)*8),
			})
			continue
		}
		if ips, err := net.LookupIP(str); err == nil {
			for _, ip := range ips {
				Passlist = append(Passlist, net.IPNet{
					IP:   ip,
					Mask: net.CIDRMask(len(ip)*8, len(ip)*8),
				})
			}
			continue
		}
		Log.Infof("white list entry '%s' does not recognized", str)
	}
	return
}

// The End.
