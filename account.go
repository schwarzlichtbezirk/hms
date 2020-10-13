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

// Share description for json-file.
type Share struct {
	Name string `json:"name"`
	Pref string `json:"pref"`
	Path string `json:"path"`
}

type Account struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`

	Roots  []string `json:"roots"`  // root directories list
	Hidden []string `json:"hidden"` // patterns for hidden files

	Shares     []Share           `json:"shares"`
	sharespath map[string]string // shares prefix by system path
	sharespref map[string]string // shares system path by prefix

	mux sync.RWMutex
}

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
func (acc *Account) ScanRoots() []ShareKit {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	var drvs = make([]ShareKit, len(acc.Roots), len(acc.Roots))
	for i, root := range acc.Roots {
		var dk DriveKit
		dk.Setup(root)
		dk.Scan(root)
		var sk = ShareKit{&dk, root, ""}
		acc.SetupPref(&sk, root)
		drvs[i] = sk
	}
	return drvs
}

// Recreates shares maps, puts share property to cache.
func (acc *Account) UpdateShares() {
	acc.mux.Lock()

	acc.sharespath = map[string]string{}
	acc.sharespref = map[string]string{}
	for _, shr := range acc.Shares {
		var shr = shr
		var err error
		var fi os.FileInfo
		if fi, err = os.Stat(shr.Path); err != nil {
			defer Log.Printf("id%d: can not create share '%s' on path '%s'", acc.ID, shr.Pref, shr.Path)
			continue
		}

		acc.sharespath[shr.Path] = shr.Pref
		acc.sharespref[shr.Pref] = shr.Path
		defer MakeProp(shr.Path, fi) // put prop to cache
		defer Log.Printf("id%d: created share '%s' on path '%s'", acc.ID, shr.Pref, shr.Path)
	}

	acc.mux.Unlock()
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
func (acc *Account) MakeShare(name, path string) (pref string) {
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

	if fit && acc.AddShare(name, pref, path) {
		return
	}
	for i := 0; !acc.AddShare(name, makerandstr(4), path); i++ {
		if i > 1000 {
			panic("can not generate share prefix")
		}
	}
	return
}

// Add share with given prefix.
func (acc *Account) AddShare(name, pref, path string) bool {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	if _, ok := acc.sharespref[pref]; !ok {
		acc.Shares = append(acc.Shares, Share{
			Name: name,
			Pref: pref,
			Path: path,
		})
		acc.sharespath[path] = pref
		acc.sharespref[pref] = path
		return true
	}
	return false
}

// Delete share by given prefix.
func (acc *Account) DelSharePref(pref string) bool {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	if path, ok := acc.sharespref[pref]; ok {
		for i, shr := range acc.Shares {
			if shr.Pref == pref {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		return true
	}
	return false
}

// Delete share by given shared path.
func (acc *Account) DelSharePath(path string) bool {
	acc.mux.Lock()
	defer acc.mux.Unlock()

	if pref, ok := acc.sharespath[path]; ok {
		for i, shr := range acc.Shares {
			if shr.Path == path {
				acc.Shares = append(acc.Shares[:i], acc.Shares[i+1:]...)
			}
		}
		delete(acc.sharespath, path)
		delete(acc.sharespref, pref)
		return true
	}
	return false
}

func (acc *Account) SetupPref(sk *ShareKit, path string) {
	acc.mux.RLock()
	defer acc.mux.RUnlock()

	if pref, ok := acc.sharespath[path]; ok {
		sk.SetPref(pref)
	}
}

// Splits given share path to share prefix and remained suffix.
func splitprefsuff(shrpath string) (string, string) {
	for i, c := range shrpath {
		if c == '/' || c == '\\' {
			return shrpath[:i], shrpath[i+1:]
		} else if c == ':' { // prefix can not be with colons
			return "", shrpath
		}
	}
	return shrpath, "" // root of share
}

// Brings share path to local file path.
func (acc *Account) GetSharePath(shrpath string) string {
	var pref, suff = splitprefsuff(shrpath)
	if pref == "" {
		return shrpath
	}

	acc.mux.RLock()
	defer acc.mux.RUnlock()

	if path, ok := acc.sharespref[pref]; ok {
		return path + suff
	}
	return shrpath
}

// Brings share path to local file path and signal that it shared.
func (acc *Account) CheckSharePath(shrpath string) (string, bool) {
	var pref, suff = splitprefsuff(shrpath)

	acc.mux.RLock()
	defer acc.mux.RUnlock()

	if pref == "" {
		var shared bool
		for _, syspath := range acc.sharespref {
			if strings.HasPrefix(shrpath, syspath) {
				shared = true
				break
			}
		}
		return shrpath, shared
	}

	if syspath, ok := acc.sharespref[pref]; ok {
		return syspath + suff, true
	}
	return shrpath, false
}

// Reads directory with given share path and returns ShareKit for each entry.
func (acc *Account) Readdir(shrpath string) (ret []ShareKit, err error) {
	var syspath = acc.GetSharePath(shrpath)
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
			var spath = shrpath + fi.Name()
			if fi.IsDir() {
				fpath += "/"
				spath += "/"
			}
			if !acc.IsHidden(fpath) {
				var sk = ShareKit{MakeProp(fpath, fi), spath, ""}
				acc.SetupPref(&sk, fpath)
				ret = append(ret, sk)
				fgrp[typetogroup[sk.Prop.Type()]]++
			}
		}
	}

	if dk, ok := MakeProp(syspath, di).(*DirKit); ok {
		dk.Scan = UnixJSNow()
		dk.FGrp = fgrp
	}

	return
}

// The End.
