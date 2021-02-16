package hms

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefHidden is default hidden path templates.
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
	// FPAnone - profile have no any access to specified file path.
	FPAnone = 0
	// FPAadmin - only authorized access to specified file path.
	FPAadmin = 1
	// FPAshare - access to specified file path is shared.
	FPAshare = 2
)

// CatGrp indicates access to each file group.
type CatGrp [FGnum]bool

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (cg *CatGrp) IsZero() bool {
	for _, v := range cg {
		if v {
			return false
		}
	}
	return true
}

// SetAll sets all elements to given boolean value.
func (cg *CatGrp) SetAll(v bool) {
	for i := range cg {
		cg[i] = v
	}
}

// Profile contains access configuration to resources.
type Profile struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`

	Roots  []string `json:"roots"`  // root directories list
	Hidden []string `json:"hidden"` // patterns for hidden files
	Shares []string `json:"shares"`

	// private shares data
	sharepuid map[string]string // share/puid key/values
	puidshare map[string]string // puid/share key/values
	ctgrshare CatGrp

	mux sync.RWMutex
}

// Profiles is the list of Profile structures.
type Profiles struct {
	list []*Profile
	mux  sync.RWMutex
}

// NewProfile make new profile and insert it to the list.
func (pl *Profiles) NewProfile(login, password string) *Profile {
	var prf = &Profile{
		Login:     login,
		Password:  password,
		Roots:     []string{},
		Hidden:    []string{},
		Shares:    []string{},
		sharepuid: map[string]string{},
		puidshare: map[string]string{},
	}
	if len(pl.list) > 0 {
		prf.ID = pl.list[len(pl.list)-1].ID + 1
	}

	pl.Insert(prf)
	return prf
}

// ByID finds profile with given identifier.
func (pl *Profiles) ByID(prfid int) *Profile {
	pl.mux.RLock()
	defer pl.mux.RUnlock()
	for _, prf := range pl.list {
		if prf.ID == prfid {
			return prf
		}
	}
	return nil
}

// ByLogin finds profile with given login.
func (pl *Profiles) ByLogin(login string) *Profile {
	pl.mux.RLock()
	defer pl.mux.RUnlock()
	for _, prf := range pl.list {
		if prf.Login == login {
			return prf
		}
	}
	return nil
}

// Insert new profile to the list.
func (pl *Profiles) Insert(prf *Profile) {
	pl.mux.Lock()
	defer pl.mux.Unlock()
	pl.list = append(pl.list, prf)
}

// Delete profile with "prfid" identifier from the list.
func (pl *Profiles) Delete(prfid int) bool {
	pl.mux.RLock()
	defer pl.mux.RUnlock()
	for i, prf := range pl.list {
		if prf.ID == prfid {
			pl.list = append(pl.list[:i], pl.list[i+1:]...)
			return true
		}
	}
	return false
}

// Profiles list.
var prflist Profiles

// SetDefaultHidden sest hidden files array to default predefined list.
func (prf *Profile) SetDefaultHidden() {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	prf.Hidden = make([]string, len(DefHidden))
	copy(prf.Hidden, DefHidden)
}

// IsHidden do check up that file path is in hidden list.
func (prf *Profile) IsHidden(fpath string) bool {
	var matched bool
	var kpath = strings.TrimSuffix(strings.ToLower(filepath.ToSlash(fpath)), "/")

	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, pattern := range prf.Hidden {
		if matched, _ = filepath.Match(pattern, kpath); matched {
			break
		}
	}
	return matched
}

// RootIndex returns index of given path in roots list or -1 if not found.
func (prf *Profile) RootIndex(path string) int {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for i, root := range prf.Roots {
		if root == path {
			return i
		}
	}
	return -1
}

// FindRoots scan all available drives installed on local machine.
func (prf *Profile) FindRoots() {
	const windisks = "CDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, d := range windisks {
		var root = string(d) + ":/"
		if _, err := os.Stat(root); err == nil {
			if prf.RootIndex(root) < 0 {
				prf.mux.Lock()
				prf.Roots = append(prf.Roots, root)
				prf.mux.Unlock()
			}
		}
	}
}

// ScanRoots scan drives from roots list.
func (prf *Profile) ScanRoots() []Pather {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var lst = make([]Pather, len(prf.Roots))
	for i, path := range prf.Roots {
		var dk DriveKit
		dk.Setup(path)
		dk.Scan(path)
		lst[i] = &dk
	}
	return lst
}

// ScanShares scan actual shares from shares list.
func (prf *Profile) ScanShares() []Pather {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var lst []Pather
	for _, path := range prf.Shares {
		if prop, err := propcache.Get(path); err == nil {
			lst = append(lst, prop.(Pather))
		}
	}
	return lst
}

// Private function to update profile shares private data.
func (prf *Profile) updateGrp() {
	var is = func(path string) bool {
		var _, ok = prf.sharepuid[path]
		return ok
	}

	var all = is(CPdrives)
	var media = is(CPmedia)
	prf.ctgrshare = CatGrp{
		all,
		all || is(CPvideo) || media,
		all || is(CPaudio) || media,
		all || is(CPimage) || media,
		all || is(CPbooks),
		all || is(CPtexts),
		all,
		all,
	}
}

// UpdateShares recreates shares maps, puts share property to cache.
func (prf *Profile) UpdateShares() {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	prf.sharepuid = map[string]string{}
	prf.puidshare = map[string]string{}
	for _, shr := range prf.Shares {
		var syspath = shr
		if prop, err := propcache.Get(syspath); err == nil {
			var puid = prop.(Pather).PUID()
			prf.sharepuid[syspath] = puid
			prf.puidshare[puid] = syspath
			Log.Printf("id%d: shared '%s' as %s", prf.ID, syspath, puid)
		} else {
			Log.Printf("id%d: can not share '%s'", prf.ID, syspath)
		}
	}
	prf.updateGrp()
}

// IsShared checks that syspath is become in any share.
func (prf *Profile) IsShared(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, path := range prf.Shares {
		if path == syspath {
			return true
		}
	}
	return false
}

// IsRooted checks that syspath is become in any root.
func (prf *Profile) IsRooted(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, path := range prf.Roots {
		if path == syspath {
			return true
		}
	}
	return false
}

// AddShare adds share with given path unigue identifier.
func (prf *Profile) AddShare(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	var puid = pathcache.Cache(syspath)
	if _, ok := prf.puidshare[puid]; !ok {
		prf.Shares = append(prf.Shares, syspath)
		prf.sharepuid[syspath] = puid
		prf.puidshare[puid] = syspath
		prf.updateGrp()
		return true
	}
	return false
}

// DelShare deletes share by given path unigue identifier.
func (prf *Profile) DelShare(puid string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	if syspath, ok := prf.puidshare[puid]; ok {
		for i, shr := range prf.Shares {
			if shr == syspath {
				prf.Shares = append(prf.Shares[:i], prf.Shares[i+1:]...)
			}
		}
		delete(prf.sharepuid, syspath)
		delete(prf.puidshare, puid)
		prf.updateGrp()
		return true
	}
	return false
}

// GetSharePath brings system path to largest share path.
func (prf *Profile) GetSharePath(syspath string, isadmin bool) (shrpath string, base string, cg CatGrp) {
	var concat = func() {
		var pref, suff = pathcache.Cache(base), syspath[len(base):]
		if len(suff) > 0 && suff[0] != '/' {
			shrpath = pref + "/" + suff
		} else {
			shrpath = pref + suff
		}
	}

	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, path := range prf.Shares {
		if strings.HasPrefix(syspath, path) {
			if len(path) > len(base) {
				base = path
			}
		}
	}
	if len(base) > 0 {
		concat()
		cg.SetAll(true)
		return
	}

	for _, path := range prf.Roots {
		if strings.HasPrefix(syspath, path) {
			if len(path) > len(base) {
				base = path
			}
		}
	}
	if len(base) > 0 {
		concat()
		if isadmin {
			cg.SetAll(true)
		} else {
			cg = prf.ctgrshare
		}
		return
	}

	shrpath = syspath
	return
}

// PathAccess returns file group access state for given file path.
func (prf *Profile) PathAccess(syspath string, isadmin bool) (cg CatGrp) {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, path := range prf.Shares {
		if strings.HasPrefix(syspath, path) {
			cg.SetAll(true)
			return
		}
	}
	for _, path := range prf.Roots {
		if strings.HasPrefix(syspath, path) {
			if isadmin {
				cg.SetAll(true)
			} else {
				cg = prf.ctgrshare
			}
			return
		}
	}
	return
}

// PathAdmin returns whether profile has admin access to file path or category path.
func (prf *Profile) PathAdmin(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, path := range prf.Shares {
		if strings.HasPrefix(syspath, path) {
			return true
		}
	}
	for _, path := range prf.Roots {
		if strings.HasPrefix(syspath, path) {
			return true
		}
	}
	for _, path := range CatPath {
		if path == syspath {
			return true
		}
	}
	return false
}

// Readdir reads directory with given system path and returns Pather for each entry.
func (prf *Profile) Readdir(syspath string, cg *CatGrp) (ret []Pather, err error) {
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

	var fgrp = FileGrp{}

	for _, fi := range fis {
		if fi != nil {
			var fpath = syspath + fi.Name()
			if fi.IsDir() {
				fpath += "/"
			}
			if !prf.IsHidden(fpath) {
				var prop = CacheProp(fpath, fi).(Pather)
				var grp = typetogroup[prop.Type()]
				if cg[grp] {
					ret = append(ret, prop)
				}
				fgrp[grp]++
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
