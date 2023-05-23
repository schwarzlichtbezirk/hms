package hms

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
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

// JoinPath performs concatenation of URL address with path elements.
func JoinPath(elem ...string) string {
	var fpath = elem[0]
	if i := strings.Index(fpath, "://"); i != -1 {
		var pref, suff string
		if j := strings.Index(fpath[i+3:], "/"); j != -1 {
			pref, suff = fpath[:i+3+j], fpath[i+3+j:]
		} else {
			pref, suff = fpath, "/"
		}
		elem[0] = suff
		return pref + path.Join(elem...)
	} else {
		return path.Join(elem...)
	}
}

// UnfoldPath brings any share path to system file path.
func UnfoldPath(session *Session, shrpath string) (syspath string, puid Puid_t, err error) {
	var pref, suff string
	if i := strings.IndexRune(shrpath, '/'); i != -1 {
		pref, suff = shrpath[:i], path.Clean(shrpath[i+1:])
		if !fs.ValidPath(suff) { // prevent to modify original path
			err = ErrPathOut
			return
		}
		if suff == "." {
			suff = ""
		}
	} else {
		pref = shrpath
	}
	var ok bool
	if puid, ok = CatPathKey[pref]; ok {
		if len(suff) > 0 {
			err = ErrNotSys
			return
		}
		syspath = pref
		return // category
	}
	if err = puid.Set(pref); err != nil {
		err = fmt.Errorf("can not decode PUID value: %w", err)
		return
	}
	if syspath, ok = PathStorePath(session, puid); !ok {
		err = ErrNoPath
		return
	}
	if len(suff) == 0 {
		return // whole cached path
	}
	syspath = JoinPath(syspath, suff)
	// get PUID if it not have
	puid = PathStoreCache(session, syspath)
	return // composite path
}

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
	ID       ID_t   `json:"id" yaml:"id" xml:"id,attr"`
	Login    string `json:"login" yaml:"login" xml:"login"`
	Password string `json:"password" yaml:"password" xml:"password"`

	Roots  []DiskPath `json:"local" yaml:"local" xml:"local>item"` // root directories list
	Remote []DiskPath `json:"remote" yaml:"remote" xml:"remote>item"`
	Shares []DiskPath `json:"shares" yaml:"shares" xml:"shares>item"`
	Hidden []string   `json:"hidden" yaml:"hidden" xml:"hidden>item"` // patterns for hidden files

	// private shares data
	ctgrshare CatGrp
	mux       sync.RWMutex
}

var prflist map[ID_t]*Profile
var plmux sync.RWMutex

// NewProfile make new profile and insert it to the list.
func NewProfile(login, password string) *Profile {
	var prf = &Profile{
		Login:    login,
		Password: password,
	}

	var mid ID_t
	for id := range prflist {
		if id > mid {
			mid = id
		}
	}
	prf.ID = mid + 1

	ProfileInsert(prf)
	return prf
}

// ProfileByID finds profile with given identifier.
func ProfileByID(prfid ID_t) *Profile {
	plmux.RLock()
	defer plmux.RUnlock()
	return prflist[prfid]
}

// ProfileByUser finds profile with given login.
func ProfileByUser(login string) *Profile {
	plmux.RLock()
	defer plmux.RUnlock()
	for _, prf := range prflist {
		if prf.Login == login {
			return prf
		}
	}
	return nil
}

// ProfileInsert new profile to the list.
func ProfileInsert(prf *Profile) {
	plmux.Lock()
	defer plmux.Unlock()
	prflist[prf.ID] = prf
}

// ProfileDelete profile with "prfid" identifier from the list.
func ProfileDelete(prfid ID_t) (ok bool) {
	plmux.RLock()
	defer plmux.RUnlock()
	if _, ok = prflist[prfid]; ok {
		delete(prflist, prfid)
	}
	return
}

// pathName returns label for given path.
func (prf *Profile) pathName(syspath string) string {
	if name, ok := CatNames[syspath]; ok {
		return name
	}

	for _, dp := range prf.Roots {
		if syspath == dp.Path {
			return dp.Name
		}
	}
	for _, dp := range prf.Remote {
		if syspath == dp.Path {
			return dp.Name
		}
	}
	var u, _ = url.Parse(syspath)
	if len(u.Path) > 0 {
		return path.Base(u.Path)
	}
	return u.Redacted()
}

// PathType returns type of file by given path.
func (prf *Profile) PathType(fpath string, fi fs.FileInfo) FT_t {
	if len(fpath) > 1 && fpath[0] == '<' && fpath[len(fpath)-1] == '>' {
		return FTctgr
	}
	if prf.IsCloud(fpath) {
		return FTcld
	}
	if prf.IsLocal(fpath) {
		return FTdrv
	}
	if fi != nil && fi.IsDir() {
		return FTdir
	}
	return FTfile
}

func (prf *Profile) GetPathGroup(fpath string, fi fs.FileInfo) (grp FG_t) {
	if prf.PathType(fpath, fi) != FTfile {
		return FGgroup
	}
	return GetFileGroup(fpath)
}

// IsHidden do check up that file path is in hidden list.
func (prf *Profile) IsHidden(fpath string) bool {
	var matched bool
	var kpath = strings.ToLower(fpath)

	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var name = path.Base(kpath)
	for _, pattern := range prf.Hidden {
		if strings.HasPrefix(pattern, "**/") {
			if matched, _ = path.Match(pattern[3:], name); matched {
				return true
			}
		} else if strings.HasPrefix(pattern, "?:/") {
			for _, dp := range prf.Roots {
				if PathStarts(kpath, strings.ToLower(dp.Path)) {
					if matched, _ = path.Match(pattern[3:], kpath[len(dp.Path):]); matched {
						return true
					}
				}
			}
		} else {
			if matched, _ = path.Match(pattern, kpath); matched {
				return true
			}
		}
	}
	return false
}

// IsLocal checks whether file path is disk root path.
func (prf *Profile) IsLocal(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, dp := range prf.Roots {
		if dp.Path == syspath {
			return true
		}
	}
	return false
}

// IsCloud checks whether file path is cloud root path.
func (prf *Profile) IsCloud(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, dp := range prf.Remote {
		if dp.Path == syspath {
			return true
		}
	}
	return false
}

// IsShared checks that syspath is become in any share.
func (prf *Profile) IsShared(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()
	for _, dp := range prf.Shares {
		if dp.Path == syspath {
			return true
		}
	}
	return false
}

// IsRemote returns true is resource is hosted anywhere outside.
func IsRemote(syspath string) bool {
	return strings.HasPrefix(syspath, "http://") ||
		strings.HasPrefix(syspath, "https://") ||
		strings.HasPrefix(syspath, "ftp://") ||
		strings.HasPrefix(syspath, "sftp://")
}

// FindLocal scans all available drives installed on local machine.
func (prf *Profile) FindLocal() {
	switch runtime.GOOS {
	case "windows":
		const windisks = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		for i := range windisks {
			var d = windisks[i : i+1]
			var root = d + ":/" // let's disk roots will be slash-terminated always
			if _, err := os.Stat(root); err == nil {
				if !prf.IsLocal(root) {
					prf.mux.Lock()
					prf.Roots = append(prf.Roots, DiskPath{
						Path: root,
						Name: "disk " + d,
					})
					prf.mux.Unlock()
				}
			}
		}
	case "linux":
		const mnt = "/mnt"
		var files, err = os.ReadDir(mnt)
		if err != nil {
			return
		}
		for _, de := range files {
			if name := de.Name(); name != "wsl" && de.IsDir() {
				var root = filepath.Join(mnt, name)
				if _, err := os.Stat(root); err == nil {
					if !prf.IsLocal(root) {
						prf.mux.Lock()
						prf.Roots = append(prf.Roots, DiskPath{
							Path: root,
							Name: "disk " + strings.ToUpper(name),
						})
						prf.mux.Unlock()
					}
				}
			}
		}
	}
}

// ScanLocal scans paths from local roots list.
func (prf *Profile) ScanLocal(session *Session) (ret []any, err error) {
	prf.mux.RLock()
	var vfiles = append([]DiskPath{}, prf.Roots...) // make non-nil copy
	prf.mux.RUnlock()

	var dp DirProp
	if ret, dp, err = ScanFileNameList(prf, session, vfiles); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, &DirStore{
			Puid: PUIDlocal,
			Prop: dp,
		})
		return
	})

	return
}

// ScanRemote scans paths at network destination.
func (prf *Profile) ScanRemote(session *Session) (ret []any, err error) {
	prf.mux.RLock()
	var vfiles = append([]DiskPath{}, prf.Remote...) // make non-nil copy
	prf.mux.RUnlock()

	var dp DirProp
	if ret, dp, err = ScanFileNameList(prf, session, vfiles); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, &DirStore{
			Puid: PUIDremote,
			Prop: dp,
		})
		return
	})

	return
}

// ScanShares scans actual shares from shares list.
func (prf *Profile) ScanShares(session *Session) (ret []any, err error) {
	prf.mux.RLock()
	var vfiles = append([]DiskPath{}, prf.Shares...) // make non-nil copy
	prf.mux.RUnlock()

	var dp DirProp
	if ret, dp, err = ScanFileNameList(prf, session, vfiles); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, &DirStore{
			Puid: PUIDshares,
			Prop: dp,
		})
		return
	})

	return
}

// Private function to update profile shares private data.
func (prf *Profile) updateGrp() {
	var is = func(fpath string) bool {
		for _, dp := range prf.Shares {
			if dp.Path == fpath {
				return true
			}
		}
		return false
	}

	var all = is(CPlocal)
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

// AddLocal adds system path to local roots list.
func (prf *Profile) AddLocal(syspath, name string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for _, dp := range prf.Roots {
		if dp.Path == syspath {
			return false
		}
	}
	prf.Roots = append(prf.Roots, DiskPath{syspath, name})
	return true
}

// DelLocal removes system path from local roots list.
func (prf *Profile) DelLocal(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for i, dp := range prf.Roots {
		if dp.Path == syspath {
			prf.Roots = append(prf.Roots[:i], prf.Roots[i+1:]...)
			return true
		}
	}
	return false
}

// AddCloud adds path to network roots list.
func (prf *Profile) AddCloud(syspath, name string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for _, dp := range prf.Remote {
		if dp.Path == syspath {
			return false
		}
	}
	prf.Remote = append(prf.Remote, DiskPath{syspath, name})
	return true
}

// DelCloud removes path from network roots list.
func (prf *Profile) DelCloud(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for i, dp := range prf.Remote {
		if dp.Path == syspath {
			prf.Remote = append(prf.Remote[:i], prf.Remote[i+1:]...)
			return true
		}
	}
	return false
}

// AddShare adds share with given system path.
func (prf *Profile) AddShare(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for _, dp := range prf.Shares {
		if dp.Path == syspath {
			return false
		}
	}
	prf.Shares = append(prf.Shares, DiskPath{syspath, prf.pathName(syspath)})
	prf.updateGrp()
	return true
}

// DelShare deletes share by given system path.
func (prf *Profile) DelShare(syspath string) bool {
	prf.mux.Lock()
	defer prf.mux.Unlock()

	for i, dp := range prf.Shares {
		if dp.Path == syspath {
			prf.Shares = append(prf.Shares[:i], prf.Shares[i+1:]...)
			prf.updateGrp()
			return true
		}
	}
	return false
}

// GetSharePath returns path in nearest shared folder that
// contains given syspath, and its name.
func (prf *Profile) GetSharePath(session *Session, syspath string) (shrpath, shrname string) {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var base string
	for _, dp := range prf.Shares {
		if PathStarts(syspath, dp.Path) {
			if len(dp.Path) > len(base) {
				base = dp.Path
				shrname = dp.Name
			}
		}
	}
	if len(base) > 0 {
		var puid = PathStoreCache(session, base)
		shrpath = JoinPath(puid.String(), syspath[len(base):])
		return
	}
	return
}

// GetRootPath returns path to nearest root path that
// contains given syspath, and its name.
func (prf *Profile) GetRootPath(session *Session, syspath string) (rootpath, rootname string) {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	var base string
	for _, dp := range prf.Roots {
		if PathStarts(syspath, dp.Path) {
			if len(dp.Path) > len(base) {
				base = dp.Path
				rootname = dp.Name
			}
		}
	}
	if len(base) > 0 {
		var puid = PathStoreCache(session, base)
		rootpath = JoinPath(puid.String(), syspath[len(base):])
		return
	}
	for _, dp := range prf.Remote {
		if PathStarts(syspath, dp.Path) {
			if len(dp.Path) > len(base) {
				base = dp.Path
				rootname = dp.Name
			}
		}
	}
	if len(base) > 0 {
		var puid = PathStoreCache(session, base)
		rootpath = JoinPath(puid.String(), syspath[len(base):])
		return
	}
	return
}

// PathAccess returns file group access state for given file path.
func (prf *Profile) PathAccess(syspath string, isadmin bool) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, dp := range prf.Shares {
		if PathStarts(syspath, dp.Path) {
			return true
		}
	}
	for _, dp := range prf.Roots {
		if PathStarts(syspath, dp.Path) {
			if isadmin {
				return true
			} else {
				var grp = GetFileGroup(syspath)
				return prf.ctgrshare[grp]
			}
		}
	}
	for _, dp := range prf.Remote {
		if PathStarts(syspath, dp.Path) {
			if isadmin {
				return true
			} else {
				var grp = GetFileGroup(syspath)
				return prf.ctgrshare[grp]
			}
		}
	}
	if _, ok := CatPathKey[syspath]; ok {
		return isadmin
	}
	return false
}

// PathAdmin returns whether profile has admin access to file path or category path.
func (prf *Profile) PathAdmin(syspath string) bool {
	prf.mux.RLock()
	defer prf.mux.RUnlock()

	for _, dp := range prf.Shares {
		if PathStarts(syspath, dp.Path) {
			return true
		}
	}
	for _, dp := range prf.Roots {
		if PathStarts(syspath, dp.Path) {
			return true
		}
	}
	for _, dp := range prf.Remote {
		if PathStarts(syspath, dp.Path) {
			return true
		}
	}
	if _, ok := CatPathKey[syspath]; ok {
		return true
	}
	return false
}

// The End.
