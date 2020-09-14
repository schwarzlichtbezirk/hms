package hms

import (
	"os"
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
var DefAcc *Account

// Accounts list.
var AccList []*Account

// Accounts map by IDs
var AccMap map[int]*Account

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

// The End.
