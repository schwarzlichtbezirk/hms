package hms

import (
	"os"
	"strings"
	"sync"
)

var DefHidden = []string{
	"?:/System Volume Information",
	"*.sys",
	"*.tmp",
	"*.bak",
	"*/.*",
	"?:/Windows",
	"?:/WindowsApps",
	"?:/$Recycle.Bin",
	"?:/Program Files",
	"?:/Program Files (x86)",
	"?:/ProgramData",
	"?:/Recovery",
	"?:/Config.Msi",
	"*/Thumb.db",
}

// Share description for json-file.
type Share struct {
	Path string `json:"path"`
	Pref string `json:"pref"`
	Name string `json:"name"`
}

type Account struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`

	Roots  []string `json:"roots"`  // root directories list
	Hidden []string `json:"hidden"` // patterns for hidden files

	Shares     []Share           `json:"shares"`
	sharespath map[string]string // active shares by full path
	sharespref map[string]string // active shares by prefix
	shrmux     sync.RWMutex
}

// Default account for user on localhost.
var (
	DefAccID = 0
	DefAcc   *Account
)

type Accounts []*Account

func (al *Accounts) ByID(aid int) *Account {
	for _, acc := range AccList {
		if acc.ID == aid {
			return acc
		}
	}
	return nil
}

func (al *Accounts) ByLogin(login string) *Account {
	for _, acc := range AccList {
		if acc.Login == login {
			return acc
		}
	}
	return nil
}

// Accounts list.
var AccList Accounts

// Recreates shares maps, puts share property to cache.
func (acc *Account) UpdateShares() {
	acc.sharespath = map[string]string{}
	acc.sharespref = map[string]string{}
	for _, shr := range acc.Shares {
		var fi, err = os.Stat(shr.Path)
		if err != nil {
			Log.Printf("can not create share '%s' on path '%s' for id=%d", shr.Pref, shr.Path, acc.ID)
			continue
		}

		var prop = MakeProp(shr.Path, fi)
		prop.SetPref(shr.Pref)
		propcache.Set(shr.Path, prop)
		acc.sharespath[shr.Path] = shr.Pref
		acc.sharespref[shr.Pref] = shr.Path
		Log.Printf("created share '%s' on path '%s' for id=%d", shr.Pref, shr.Path, acc.ID)
	}
}

var sharecharset = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func makerandstr(n int) string {
	var l = byte(len(sharecharset))
	var str = make([]byte, n)
	randbytes(str)
	for i := 0; i < n; i++ {
		str[i] = sharecharset[str[i]%l]
	}
	return string(str)
}

// Looks for correct prefix and add share with it.
func (acc *Account) MakeShare(path string, prop FileProper) {
	var pref string
	var name = prop.Name()
	if len(name) > 8 {
		pref = name[:8]
	} else {
		pref = name
	}
	var fit = true
	for _, b := range pref {
		if (b < '0' || b > '9') && (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && b != '-' && b != '_' {
			fit = false
		}
	}

	if fit && acc.AddShare(pref, path, prop) {
		return
	}
	for i := 0; !acc.AddShare(makerandstr(4), path, prop); i++ {
		if i > 1000 {
			panic("can not generate share prefix")
		}
	}
}

// Add share with given prefix.
func (acc *Account) AddShare(pref string, path string, prop FileProper) bool {
	acc.shrmux.RLock()
	var _, ok = acc.sharespref[pref]
	acc.shrmux.RUnlock()

	if !ok {
		prop.SetPref(pref)

		acc.shrmux.Lock()
		acc.Shares = append(acc.Shares, Share{
			Path: path,
			Pref: pref,
			Name: prop.Name(),
		})
		acc.sharespath[path] = pref
		acc.sharespref[pref] = path
		acc.shrmux.Unlock()
	}
	return !ok
}

// Delete share by given prefix.
func (acc *Account) DelSharePref(pref string) bool {
	acc.shrmux.RLock()
	var path, ok = acc.sharespref[pref]
	acc.shrmux.RUnlock()

	if ok {
		if cp, err := propcache.Get(path); err == nil {
			cp.(FileProper).SetPref("")
		}

		acc.shrmux.Lock()
		for i, shr := range acc.Shares {
			if shr.Pref == pref {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		acc.shrmux.Unlock()
	}
	return ok
}

// Delete share by given shared path.
func (acc *Account) DelSharePath(path string) bool {
	acc.shrmux.RLock()
	var pref, ok = acc.sharespath[path]
	acc.shrmux.RUnlock()

	if ok {
		if cp, err := propcache.Get(path); err == nil {
			cp.(FileProper).SetPref("")
		}

		acc.shrmux.Lock()
		for i, shr := range acc.Shares {
			if shr.Path == path {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		acc.shrmux.Unlock()
	}
	return ok
}

// Returns share prefix and remained suffix
func splitprefsuff(share string) (string, string) {
	for i, c := range share {
		if c == '/' || c == '\\' {
			return share[:i], share[i+1:]
		} else if c == ':' { // prefix can not be with colons
			return "", share
		}
	}
	return share, "" // root of share
}

// Brings share path to local file path.
func (acc *Account) GetSharePath(spath string) string {
	var pref, suff = splitprefsuff(spath)
	if pref == "" {
		return spath
	}
	acc.shrmux.RLock()
	var path, ok = acc.sharespref[pref]
	acc.shrmux.RUnlock()
	if !ok {
		return spath
	}
	return path + suff
}

// Brings share path to local file path and signal that it shared.
func (acc *Account) CheckSharePath(spath string) (string, bool) {
	var pref, suff = splitprefsuff(spath)
	if pref == "" {
		var shared bool
		acc.shrmux.RLock()
		for _, fpath := range acc.sharespref {
			if strings.HasPrefix(spath, fpath) {
				shared = true
				break
			}
		}
		acc.shrmux.RUnlock()
		return spath, shared
	}
	acc.shrmux.RLock()
	var fpath, ok = acc.sharespref[pref]
	acc.shrmux.RUnlock()
	if !ok {
		return spath, false
	}
	return fpath + suff, true
}

// The End.
