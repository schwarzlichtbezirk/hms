package hms

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// UnfoldPath brings any share path to system file path.
func UnfoldPath(shrpath string) (syspath string, err error) {
	shrpath = path.Clean(shrpath)
	var pref, suff = shrpath, "."
	if i := strings.IndexRune(shrpath, '/'); i != -1 {
		pref, suff = shrpath[:i], shrpath[i+1:]
		if !fs.ValidPath(suff) { // prevent to modify original path
			err = ErrPathOut
			return
		}
	}
	var puid Puid_t
	var ok bool
	if puid, ok = CatPathKey[pref]; !ok {
		if err = puid.Set(pref); err != nil {
			err = fmt.Errorf("can not decode PUID value: %w", err)
			return
		}
		if pref, ok = PathCachePath(puid); !ok {
			err = ErrNoPath
			return
		}
	} else if suff != "." {
		err = ErrNotSys
		return
	}
	syspath = path.Join(pref, suff)
	// append slash to disk root to prevent open current dir on this disk
	if syspath[len(syspath)-1] == ':' { // syspath here is always have non zero length
		syspath += "/"
	}
	return
}

// APIHANDLER
func folderAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t   `json:"aid" yaml:"aid" xml:"aid,attr"`
		Path string `json:"path,omitempty" yaml:"path,omitempty" xml:"path,omitempty"`
		Ext  string `json:"ext,omitempty" yaml:"ext,omitempty" xml:"ext,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Pather `json:"list" yaml:"list" xml:"list>prop"`
		Skip int      `json:"skip" yaml:"skip" xml:"skip"`
		PUID Puid_t   `json:"puid" yaml:"puid" xml:"puid"`
		Path string   `json:"path" yaml:"path" xml:"path"`
		Name string   `json:"shrname" yaml:"shrname" xml:"shrname"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECfoldernodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECfoldernoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	var syspath string
	if syspath, err = UnfoldPath(ToSlash(arg.Path)); err != nil {
		WriteError400(w, r, err, AECfolderbadpath)
		return
	}

	if prf.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECfolderhidden)
		return
	}

	var shrpath, base, cg = prf.GetSharePath(syspath, auth == prf)
	if cg.IsZero() && syspath != CPshares {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECfolderaccess)
		return
	}
	ret.PUID = PathCacheSure(syspath)
	ret.Path = shrpath
	ret.Name = PathBase(base)

	var t = time.Now()
	if ret.PUID < PUIDcache {
		if auth != prf && !prf.IsShared(syspath) {
			WriteError(w, r, http.StatusForbidden, ErrNotShared, AECfoldernoshr)
			return
		}
		var catprop = func(puids []Puid_t) {
			for _, puid := range puids {
				if fpath, ok := PathCachePath(puid); ok {
					if prop, err := propcache.Get(fpath); err == nil {
						ret.List = append(ret.List, prop.(Pather))
					}
				}
			}
		}
		switch ret.PUID {
		case PUIDhome:
			for puid := Puid_t(1); puid < PUIDcache; puid++ {
				if puid == PUIDhome {
					continue
				}
				if fpath, ok := CatKeyPath[puid]; ok {
					if auth == prf || prf.IsShared(fpath) {
						if prop, err := propcache.Get(fpath); err == nil {
							ret.List = append(ret.List, prop.(Pather))
						}
					}
				}
			}
		case PUIDdrives:
			ret.List = prf.ScanRoots()
		case PUIDshares:
			ret.List = prf.ScanShares()
		case PUIDmedia:
			var puids []Puid_t
			if puids, err = DirCacheCat("video+audio+image", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDvideo:
			var puids []Puid_t
			if puids, err = DirCacheCat("video", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDaudio:
			var puids []Puid_t
			if puids, err = DirCacheCat("audio", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDimage:
			var puids []Puid_t
			if puids, err = DirCacheCat("image", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDbooks:
			var puids []Puid_t
			if puids, err = DirCacheCat("books", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDtexts:
			var puids []Puid_t
			if puids, err = DirCacheCat("texts", 0.5); err != nil {
				WriteError500(w, r, err, AECfolderdircat)
				return
			}
			catprop(puids)
		case PUIDmap:
			var n = cfg.RangeSearchAny
			gpscache.Range(func(puid Puid_t, gps *GpsInfo) bool {
				if fpath, ok := PathCachePath(puid); ok {
					if auth == prf || prf.IsShared(fpath) {
						if prop, err := propcache.Get(fpath); err == nil {
							ret.List = append(ret.List, prop.(Pather))
							n--
						}
					}
				}
				return n > 0
			})
		default:
			WriteError(w, r, http.StatusNotFound, ErrNotCat, AECfoldernotcat)
			return
		}
	} else {
		var fi fs.FileInfo
		if fi, err = StatFile(syspath); err != nil {
			WriteError500(w, r, err, AECfolderstat)
			return
		}

		var ext = arg.Ext
		if ext == "" {
			ext = GetFileExt(syspath)
		}

		if !fi.IsDir() && IsTypePlaylist(ext) {
			var file io.ReadCloser
			if file, err = OpenFile(syspath); err != nil {
				WriteError500(w, r, err, AECfolderopen)
				return
			}
			defer file.Close()

			var pl Playlist
			pl.Dest = path.Dir(syspath)
			switch ext {
			case ".m3u", ".m3u8":
				if _, err = pl.ReadM3U(file); err != nil {
					WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderm3u)
					return
				}
			case ".wpl":
				if _, err = pl.ReadWPL(file); err != nil {
					WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderwpl)
					return
				}
			case ".pls":
				if _, err = pl.ReadPLS(file); err != nil {
					WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderpls)
					return
				}
			case ".asx":
				if _, err = pl.ReadASX(file); err != nil {
					WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderasx)
					return
				}
			case ".xspf":
				if _, err = pl.ReadXSPF(file); err != nil {
					WriteError(w, r, http.StatusUnsupportedMediaType, err, AECfolderxspf)
					return
				}
			default:
				WriteError(w, r, http.StatusUnsupportedMediaType, ErrNotPlay, AECfolderformat)
				return
			}

			var prop interface{}
			for _, track := range pl.Tracks {
				var fpath = ToSlash(track.Location)
				if !prf.IsHidden(fpath) {
					var cg = prf.PathAccess(fpath, auth == prf)
					var grp = GetFileGroup(fpath)
					if cg[grp] {
						if prop, err = propcache.Get(fpath); err == nil {
							ret.List = append(ret.List, prop.(Pather))
							continue
						}
					}
				}
			}
			ret.Skip = len(pl.Tracks) - len(ret.List)
		} else {
			if ret.List, ret.Skip, err = ScanDir(syspath, &cg, func(fpath string) bool {
				return !prf.IsHidden(fpath)
			}); err != nil && len(ret.List) == 0 {
				if errors.Is(err, fs.ErrNotExist) {
					WriteError(w, r, http.StatusNotFound, err, AECfolderabsent)
				} else {
					WriteError500(w, r, err, AECfolderfail)
				}
				return
			}
		}
	}

	Log.Infof("id%d: navigate to %s, items %d, timeout %s", prf.ID, syspath, len(ret.List), time.Since(t))
	usermsg <- UsrMsg{r, "path", ret.PUID}

	WriteOK(w, r, &ret)
}

// The End.
