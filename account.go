package hms

import (
	"os"
	"path/filepath"
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
	"*/thumbs.db",
}

// File path access.
const (
	FPA_none  = 0 // account have no any access to specified file path
	FPA_admin = 1 // only authorized access to specified file path
	FPA_share = 2 // access to specified file path is shared
)

type Account struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`

	Roots  []string `json:"roots"`  // root directories list
	Hidden []string `json:"hidden"` // patterns for hidden files
	Shares []string `json:"shares"`

	// private shares data
	sharepuid map[string]string // share/puid key/values
	puidshare map[string]string // puid/share key/values
	ctgrshare [FG_num]bool
	allshared bool

	mux sync.RWMutex
}

type Accounts struct {
	list []*Account
	mux  sync.RWMutex
}

func (al *Accounts) NewAccount(login, password string) *Account {
	var acc = &Account{
		Login:     login,
		Password:  password,
		Roots:     []string{},
		Hidden:    []string{},
		Shares:    []string{},
		sharepuid: map[string]string{},
		puidshare: map[string]string{},
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
var acclist Accounts

// Set hidden files array to default predefined list.
func (acc *Account) SetDefaultHidden() {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	acc.Hidden = make([]string, len(DefHidden))
	copy(acc.Hidden, DefHidden)
}

// Check up that file path is in hidden list.
func (acc *Account) IsHidden(fpath string) bool {
	var matched bool
	var kpath = strings.TrimSuffix(strings.ToLower(filepath.ToSlash(fpath)), "/")

	acc.mux.RLock()
	defer acc.mux.RUnlock()

	for _, pattern := range acc.Hidden {
		if matched, _ = filepath.Match(pattern, kpath); matched {
			break
		}
	}
	return matched
}

// Returns index of given path in roots list or -1 if not found.
func (acc *Account) RootIndex(path string) int {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	for i, root := range acc.Roots {
		if root == path {
			return i
		}
	}
	return -1
}

// Scan all available drives installed on local machine.
func (acc *Account) FindRoots() {
	const windisks = "CDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, d := range windisks {
		var root = string(d) + ":/"
		if _, err := os.Stat(root); err == nil {
			if acc.RootIndex(root) < 0 {
				acc.mux.Lock()
				acc.Roots = append(acc.Roots, root)
				acc.mux.Unlock()
			}
		}
	}
}

// Scan drives from roots list.
func (acc *Account) ScanRoots() []Proper {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	var lst = make([]Proper, len(acc.Roots), len(acc.Roots))
	for i, path := range acc.Roots {
		var dk DriveKit
		dk.Setup(path)
		dk.Scan(path)
		lst[i] = &dk
	}
	return lst
}

// Scan actual shares from shares list.
func (acc *Account) ScanShares() []Proper {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	var lst []Proper
	for _, path := range acc.Shares {
		if prop, err := propcache.Get(path); err == nil {
			lst = append(lst, prop.(Proper))
		}
	}
	return lst
}

// Recreates shares maps, puts share property to cache.
func (acc *Account) UpdateShares() {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	acc.sharepuid = map[string]string{}
	acc.puidshare = map[string]string{}
	for _, shr := range acc.Shares {
		var syspath = shr
		if prop, err := propcache.Get(syspath); err == nil {
			var puid = prop.(Proper).PUID()
			acc.sharepuid[syspath] = puid
			acc.puidshare[puid] = syspath
			Log.Printf("id%d: shared '%s' as %s", acc.ID, syspath, puid)
		} else {
			Log.Printf("id%d: can not share '%s'", acc.ID, syspath)
		}
	}
}

// Checks that syspath is become in any share.
func (acc *Account) IsShared(syspath string) bool {
	acc.mux.RLock()
	defer acc.mux.RUnlock()
	for _, path := range acc.Shares {
		if path == syspath {
			return true
		}
	}
	return false
}

// Checks that syspath is become in any root.
func (acc *Account) IsRooted(syspath string) bool {
	acc.mux.RLock()
	defer acc.mux.RUnlock()
	for _, path := range acc.Roots {
		if path == syspath {
			return true
		}
	}
	return false
}

// Add share with given path unigue identifier.
func (acc *Account) AddShare(syspath string) bool {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	var puid = pathcache.Cache(syspath)
	if _, ok := acc.puidshare[puid]; !ok {
		acc.Shares = append(acc.Shares, syspath)
		acc.sharepuid[syspath] = puid
		acc.puidshare[puid] = syspath
		return true
	}
	return false
}

// Delete share by given path unigue identifier.
func (acc *Account) DelShare(puid string) bool {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	if syspath, ok := acc.puidshare[puid]; ok {
		for i, shr := range acc.Shares {
			if shr == syspath {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharepuid, syspath)
		delete(acc.puidshare, puid)
		return true
	}
	return false
}

// Brings system path to largest share path.
func (acc *Account) GetSharePath(syspath string) (string, string, int) {
	var base string
	var concat = func() string {
		var pref, suff = pathcache.Cache(base), syspath[len(base):]
		if len(suff) > 0 && suff[0] != '/' {
			return pref + "/" + suff
		} else {
			return pref + suff
		}
	}

	acc.mux.RLock()
	defer acc.mux.RUnlock()

	for _, path := range acc.Shares {
		if strings.HasPrefix(syspath, path) {
			if len(path) > len(base) {
				base = path
			}
		}
	}
	if len(base) > 0 {
		return concat(), base, FPA_share
	}

	for _, path := range acc.Roots {
		if strings.HasPrefix(syspath, path) {
			if len(path) > len(base) {
				base = path
			}
		}
	}
	if len(base) > 0 {
		return concat(), base, FPA_admin
	}

	return syspath, "", FPA_none
}

// Returns access state of file path, is it shared by account,
// has access only by authorization, or has no any access.
func (acc *Account) PathState(syspath string) int {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	for _, path := range acc.Shares {
		if strings.HasPrefix(syspath, path) {
			return FPA_share
		}
	}
	for _, path := range acc.Roots {
		if strings.HasPrefix(syspath, path) {
			return FPA_admin
		}
	}
	for _, path := range CatPath {
		if path == syspath {
			return FPA_admin
		}
	}
	return FPA_none
}

// Reads directory with given system path and returns Proper for each entry.
func (acc *Account) Readdir(syspath string) (ret []Proper, err error) {
	if !strings.HasSuffix(syspath, "/") {
		syspath += "/"
	}

	var di os.FileInfo
	var fis []os.FileInfo
	if func() {
		var file *os.File
		if file, err = os.Open(syspath); err != nil {
			return
		}
		defer file.Close()

		if di, err = file.Stat(); err != nil {
			return
		}
		if fis, err = file.Readdir(-1); err != nil {
			return
		}
	}(); err != nil {
		return
	}

	var fgrp = [FG_num]int{}

	for _, fi := range fis {
		if fi != nil {
			var fpath = syspath + fi.Name()
			if fi.IsDir() {
				fpath += "/"
			}
			if !acc.IsHidden(fpath) {
				var prop = CacheProp(fpath, fi)
				ret = append(ret, prop)
				fgrp[typetogroup[prop.Type()]]++
			}
		}
	}

	if dk, ok := CacheProp(syspath, di).(*DirKit); ok {
		dk.Scan = UnixJSNow()
		dk.FGrp = fgrp
		dircache.Set(dk.PUIDVal, dk.DirProp)
	}

	return
}

// The End.
