package hms

import (
	"os"
	"strings"
	"sync"
)

var DefHidden = []string{
	"?:/system volume information",
	"*.sys",
	"*.tmp",
	"*.bak",
	"*/.*",
	"?:/windows",
	"?:/windowsapps",
	"?:/$recycle.bin",
	"?:/program files",
	"?:/program files (x86)",
	"?:/programdata",
	"?:/recovery",
	"?:/config.msi",
	"*/thumb.db",
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

	mux sync.RWMutex
}

// Default account for user on localhost.
var DefAccID = 0

type Accounts struct {
	list []*Account
	mux  sync.RWMutex
}

func (al *Accounts) NewAccount(login, password string) *Account {
	var acc = &Account{
		Login:      login,
		Password:   password,
		Roots:      []string{},
		Hidden:     []string{},
		Shares:     []Share{},
		sharespath: map[string]string{},
		sharespref: map[string]string{},
	}
	if len(al.list) > 0 {
		acc.ID = al.list[len(al.list)-1].ID + 1
	}

	al.Insert(acc)
	return acc
}

func (al *Accounts) ByID(aid int) *Account {
	al.mux.RLock()
	defer al.mux.RUnlock()
	for _, acc := range al.list {
		if acc.ID == aid {
			return acc
		}
	}
	return nil
}

func (al *Accounts) ByLogin(login string) *Account {
	al.mux.RLock()
	defer al.mux.RUnlock()
	for _, acc := range al.list {
		if acc.Login == login {
			return acc
		}
	}
	return nil
}

func (al *Accounts) Insert(acc *Account) {
	al.mux.Lock()
	defer al.mux.Unlock()
	al.list = append(al.list, acc)
}

func (al *Accounts) Delete(aid int) bool {
	al.mux.RLock()
	defer al.mux.RUnlock()
	for i, acc := range al.list {
		if acc.ID == aid {
			al.list = append(al.list[:i], al.list[i+1:]...)
			return true
		}
	}
	return false
}

// Accounts list.
var AccList Accounts

// Set hidden files array to default predefined list.
func (acc *Account) SetDefaultHidden() {
	acc.Hidden = make([]string, len(DefHidden))
	copy(acc.Hidden, DefHidden)
}

// Scan all available drives installed on local machine.
func (acc *Account) FindRoots() {
	const windisks = "CDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, d := range windisks {
		var path = string(d) + ":/"
		if _, err := os.Stat(path); err == nil {
			var found = false
			for _, root := range acc.Roots {
				if root == path {
					found = true
					break
				}
			}
			if !found {
				acc.Roots = append(acc.Roots, path)
			}
		}
	}
}

// Scan drives from roots list.
func (acc *Account) ScanRoots() (drvs []FileProper) {
	drvs = make([]FileProper, len(acc.Roots), len(acc.Roots))
	for i, root := range acc.Roots {
		_, err := os.Stat(root)
		var dk DriveKit
		dk.Setup(root, err != nil)
		drvs[i] = &dk
	}
	return
}

// Recreates shares maps, puts share property to cache.
func (acc *Account) UpdateShares() {
	acc.sharespath = map[string]string{}
	acc.sharespref = map[string]string{}
	for _, shr := range acc.Shares {
		var fi, err = os.Stat(shr.Path)
		if err != nil {
			Log.Printf("id%d: can not create share '%s' on path '%s'", acc.ID, shr.Pref, shr.Path)
			continue
		}

		var prop = MakeProp(shr.Path, fi)
		prop.SetPref(shr.Pref)
		propcache.Set(shr.Path, prop)
		acc.sharespath[shr.Path] = shr.Pref
		acc.sharespref[shr.Pref] = shr.Path
		Log.Printf("id%d: created share '%s' on path '%s'", acc.ID, shr.Pref, shr.Path)
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
	acc.mux.RLock()
	var _, ok = acc.sharespref[pref]
	acc.mux.RUnlock()

	if !ok {
		prop.SetPref(pref)

		acc.mux.Lock()
		acc.Shares = append(acc.Shares, Share{
			Path: path,
			Pref: pref,
			Name: prop.Name(),
		})
		acc.sharespath[path] = pref
		acc.sharespref[pref] = path
		acc.mux.Unlock()
	}
	return !ok
}

// Delete share by given prefix.
func (acc *Account) DelSharePref(pref string) bool {
	acc.mux.RLock()
	var path, ok = acc.sharespref[pref]
	acc.mux.RUnlock()

	if ok {
		if cp, err := propcache.Get(path); err == nil {
			cp.(FileProper).SetPref("")
		}

		acc.mux.Lock()
		for i, shr := range acc.Shares {
			if shr.Pref == pref {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		acc.mux.Unlock()
	}
	return ok
}

// Delete share by given shared path.
func (acc *Account) DelSharePath(path string) bool {
	acc.mux.RLock()
	var pref, ok = acc.sharespath[path]
	acc.mux.RUnlock()

	if ok {
		if cp, err := propcache.Get(path); err == nil {
			cp.(FileProper).SetPref("")
		}

		acc.mux.Lock()
		for i, shr := range acc.Shares {
			if shr.Path == path {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		acc.mux.Unlock()
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
	acc.mux.RLock()
	var path, ok = acc.sharespref[pref]
	acc.mux.RUnlock()
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
		acc.mux.RLock()
		for _, fpath := range acc.sharespref {
			if strings.HasPrefix(spath, fpath) {
				shared = true
				break
			}
		}
		acc.mux.RUnlock()
		return spath, shared
	}
	acc.mux.RLock()
	var fpath, ok = acc.sharespref[pref]
	acc.mux.RUnlock()
	if !ok {
		return spath, false
	}
	return fpath + suff, true
}

// The End.
