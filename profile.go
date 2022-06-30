package hms

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// DefHidden is default hidden path templates.
var DefHidden = []string{
	"**/*.sys",
	"**/*.tmp",
	"**/*.bak",
	"**/.*",
	"**/Thumbs.db",
	"?:/System Volume Information",
	"?:/Windows",
	"?:/WindowsApps",
	"?:/$Recycle.Bin",
	"?:/Program Files",
	"?:/Program Files (x86)",
	"?:/ProgramData",
	"?:/Recovery",
	"?:/Config.Msi",
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

var ToSlash = filepath.ToSlash

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
	ID       ID_t   `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`

	Roots  []string `json:"roots"`  // root directories list
	Hidden []string `json:"hidden"` // patterns for hidden files
	Shares []string `json:"shares"`

	// private shares data
	sharepuid map[string]Puid_t // share/puid key/values
	puidshare map[Puid_t]string // puid/share key/values
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
		sharepuid: map[string]Puid_t{},
		puidshare: map[Puid_t]string{},
	}
	if len(pl.list) > 0 {
		prf.ID = pl.list[len(pl.list)-1].ID + 1
	}

	pl.Insert(prf)
	return prf
}

// ByID finds profile with given identifier.
func (pl *Profiles) ByID(prfid ID_t) *Profile {
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
func (pl *Profiles) Delete(prfid ID_t) bool {
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
	var kpath = strings.ToLower(ToSlash(fpath))

	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var name = path.Base(kpath)
	for _, pattern := range prf.Hidden {
		if strings.HasPrefix(pattern, "**/") {
			if matched, _ = path.Match(pattern[3:], name); matched {
				break
			}
		} else {
			if matched, _ = path.Match(pattern, kpath); matched {
				break
			}
		}
	}
	return matched
}

// RootIndex returns index of given path in roots list or -1 if not found.
func (prf *Profile) RootIndex(fpath string) int {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for i, root := range prf.Roots {
		if root == fpath {
			return i
		}
	}
	return -1
}

// FindRoots scan all available drives installed on local machine.
func (prf *Profile) FindRoots() {
	const windisks = "CDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, d := range windisks {
		var root = string(d) + ":/" // let's disk roots will be slash-terminated always
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
	for i, fpath := range prf.Roots {
		var dk DriveKit
		dk.Setup(fpath)
		dk.Scan(fpath)
		lst[i] = &dk
	}
	return lst
}

// ScanShares scan actual shares from shares list.
func (prf *Profile) ScanShares() []Pather {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var lst []Pather
	for _, fpath := range prf.Shares {
		if prop, err := propcache.Get(fpath); err == nil {
			lst = append(lst, prop.(Pather))
		}
	}
	return lst
}

// Private function to update profile shares private data.
func (prf *Profile) updateGrp() {
	var is = func(fpath string) bool {
		var _, ok = prf.sharepuid[fpath]
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

	prf.sharepuid = map[string]Puid_t{}
	prf.puidshare = map[Puid_t]string{}
	for _, shr := range prf.Shares {
		var syspath = shr
		if prop, err := propcache.Get(syspath); err == nil {
			var puid = prop.(Puider).PUID()
			prf.sharepuid[syspath] = puid
			prf.puidshare[puid] = syspath
			Log.Infof("id%d: shared '%s' as %s", prf.ID, syspath, puid)
		} else {
			Log.Infof("id%d: can not share '%s'", prf.ID, syspath)
		}
	}
	prf.updateGrp()
}

// IsShared checks that syspath is become in any share.
func (prf *Profile) IsShared(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, fpath := range prf.Shares {
		if fpath == syspath {
			return true
		}
	}
	return false
}

// IsRooted checks that syspath is become in any root.
func (prf *Profile) IsRooted(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, fpath := range prf.Roots {
		if fpath == syspath {
			return true
		}
	}
	return false
}

// AddShare adds share with given path unigue identifier.
func (prf *Profile) AddShare(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	var puid = syspathcache.Cache(syspath)
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
func (prf *Profile) DelShare(puid Puid_t) bool {
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
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, fpath := range prf.Shares {
		if strings.HasPrefix(syspath, fpath) {
			if len(fpath) > len(base) {
				base = fpath
			}
		}
	}
	if len(base) > 0 {
		shrpath = path.Join(syspathcache.Cache(base).String(), syspath[len(base):])
		cg.SetAll(true)
		return
	}

	for _, fpath := range prf.Roots {
		if strings.HasPrefix(syspath, fpath) {
			if len(fpath) > len(base) {
				base = fpath
			}
		}
	}
	if len(base) > 0 {
		shrpath = path.Join(syspathcache.Cache(base).String(), syspath[len(base):])
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

	for _, fpath := range prf.Shares {
		if strings.HasPrefix(syspath, fpath) {
			cg.SetAll(true)
			return
		}
	}
	for _, fpath := range prf.Roots {
		if strings.HasPrefix(syspath, fpath) {
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

	for _, fpath := range prf.Shares {
		if strings.HasPrefix(syspath, fpath) {
			return true
		}
	}
	for _, fpath := range prf.Roots {
		if strings.HasPrefix(syspath, fpath) {
			return true
		}
	}
	if _, ok := CatPathKey[syspath]; ok {
		return true
	}
	return false
}

// The End.
