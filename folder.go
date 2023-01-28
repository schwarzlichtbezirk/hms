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

	"github.com/gorilla/mux"
)

var catcolumn = map[Puid_t]string{
	PUIDmedia: "video+audio+image",
	PUIDvideo: "video",
	PUIDaudio: "audio",
	PUIDimage: "image",
	PUIDbooks: "books",
	PUIDtexts: "texts",
}

// UnfoldPath brings any share path to system file path.
func UnfoldPath(session *Session, shrpath string) (syspath string, puid Puid_t, err error) {
	shrpath = path.Clean(shrpath)
	var pref, suff = shrpath, "."
	if i := strings.IndexRune(shrpath, '/'); i != -1 {
		pref, suff = shrpath[:i], shrpath[i+1:]
		if !fs.ValidPath(suff) { // prevent to modify original path
			err = ErrPathOut
			return
		}
	}
	var ok bool
	if puid, ok = CatPathKey[pref]; !ok {
		if err = puid.Set(pref); err != nil {
			err = fmt.Errorf("can not decode PUID value: %w", err)
			return
		}
		if pref, ok = PathStorePath(session, puid); !ok {
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
	// get PUID if it not have
	if suff != "." {
		puid = PathStoreCache(session, syspath)
	}
	return
}

// APIHANDLER
func folderAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path,omitempty" yaml:"path,omitempty" xml:"path,omitempty,attr"`
		Ext  string `json:"ext,omitempty" yaml:"ext,omitempty" xml:"ext,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []any  `json:"list" yaml:"list" xml:"list>prop"`
		Skip int    `json:"skip" yaml:"skip" xml:"skip,attr"`
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		Path string `json:"path" yaml:"path" xml:"path,attr"`
		Name string `json:"shrname" yaml:"shrname" xml:"shrname,attr"`

		HasHome bool `json:"hashome" yaml:"hashome" xml:"hashome,attr"`
	}

	// get arguments
	var vars = mux.Vars(r)
	var aid ID_t
	if aid, err = ParseID(vars["aid"]); err != nil {
		WriteError400(w, r, err, AECfoldernoaid)
		return
	}
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECfoldernodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECfoldernoacc)
		return
	}
	var uid ID_t
	if uid, err = GetAuth(r); err != nil {
		WriteRet(w, r, http.StatusUnauthorized, err)
		return
	}

	var syspath string
	var puid Puid_t
	if syspath, puid, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		WriteError400(w, r, err, AECfolderbadpath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECfolderhidden)
		return
	}

	var shrpath, base, cg = acc.GetSharePath(session, syspath, uid == aid)
	if cg.IsZero() && syspath != CPshares {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECfolderaccess)
		return
	}
	ret.PUID = puid
	ret.Path = shrpath
	if puid0, ok := CatPathKey[base]; ok {
		ret.Name = CatNames[puid0]
	} else {
		ret.Name = path.Base(base)
	}

	if uid == aid {
		ret.HasHome = true
	} else if acc.IsShared(CPhome) {
		for _, fpath := range CatKeyPath {
			if fpath == CPhome {
				continue
			}
			if acc.IsShared(fpath) {
				ret.HasHome = true
				break
			}
		}
	}

	var t = time.Now()
	if puid < PUIDcache {
		if uid != aid && !acc.IsShared(syspath) {
			WriteError(w, r, http.StatusForbidden, ErrNotShared, AECfoldernoshr)
			return
		}
		switch puid {
		case PUIDhome:
			var vfiles []string
			for puid := Puid_t(1); puid < PUIDcache; puid++ {
				if puid == PUIDhome {
					continue
				}
				if fpath, ok := CatKeyPath[puid]; ok {
					if uid == aid || acc.IsShared(fpath) {
						vfiles = append(vfiles, fpath)
					}
				}
			}
			var lstp DirProp
			if ret.List, lstp, err = ScanFileNameList(acc, session, vfiles); err != nil {
				WriteError500(w, r, err, AECfolderhome)
				return
			}
			go SqlSession(func(session *Session) (res any, err error) {
				DirStoreSet(session, &DirStore{
					Puid: puid,
					Prop: lstp,
				})
				return
			})
		case PUIDdrives:
			if ret.List, err = acc.ScanRoots(session); err != nil {
				WriteError500(w, r, err, AECfolderdrives)
				return
			}
		case PUIDshares:
			if ret.List, err = acc.ScanShares(session); err != nil {
				WriteError500(w, r, err, AECfoldershares)
				return
			}
		case PUIDmedia, PUIDvideo, PUIDaudio, PUIDimage, PUIDbooks, PUIDtexts:
			if ret.List, err = ScanCat(acc, session, puid, catcolumn[puid], 0.5); err != nil {
				WriteError500(w, r, err, AECfoldermedia)
				return
			}
		case PUIDmap:
			var n = 0
			var vfiles []fs.FileInfo // verified file infos
			var vpaths []string      // verified paths
			gpscache.Range(func(puid Puid_t, gps GpsInfo) bool {
				if fpath, ok := pathcache.GetDir(puid); ok {
					if !acc.IsHidden(fpath) && acc.PathAccess(fpath, uid == aid) {
						if fi, _ := StatFile(fpath); fi != nil {
							vfiles = append(vfiles, fi)
							vpaths = append(vpaths, fpath)
							n++
						}
					}
				}
				return cfg.RangeSearchAny <= 0 || n < cfg.RangeSearchAny
			})
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths); err != nil {
				WriteError500(w, r, err, AECfoldermap)
				return
			}
		default:
			WriteError(w, r, http.StatusNotFound, ErrNoCat, AECfoldernocat)
			return
		}
	} else {
		var fi fs.FileInfo
		if fi, err = StatFile(syspath); err != nil {
			WriteError500(w, r, err, AECfolderstat)
			return
		}

		var ext = arg.Ext
		if ext == "" && !fi.IsDir() {
			ext = GetFileExt(syspath)
		}

		if fi.IsDir() || IsTypeISO(ext) {
			if ret.List, ret.Skip, err = ScanDir(acc, session, syspath, &cg); err != nil && len(ret.List) == 0 {
				if errors.Is(err, fs.ErrNotExist) {
					WriteError(w, r, http.StatusNotFound, err, AECfolderabsent)
				} else {
					WriteError500(w, r, err, AECfolderfail)
				}
				return
			}
		} else if IsTypePlaylist(ext) {
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

			var vfiles []fs.FileInfo // verified file infos
			var vpaths []string      // verified paths
			for _, track := range pl.Tracks {
				var fpath = ToSlash(track.Location)
				if !acc.IsHidden(fpath) && acc.PathAccess(fpath, uid == aid) {
					if fi, _ := StatFile(fpath); fi != nil {
						vfiles = append(vfiles, fi)
						vpaths = append(vpaths, fpath)
					}
				}
			}
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths); err != nil {
				WriteError500(w, r, err, AECfoldertracks)
				return
			}
			ret.Skip = len(pl.Tracks) - len(ret.List)
		}
	}

	var latency = time.Since(t)
	Log.Infof("id%d: navigate to %s, items %d, timeout %s", acc.ID, syspath, len(ret.List), latency)
	if cid, err := GetCID(r); err == nil {
		go xormUserlog.InsertOne(&OpenStore{
			CID:     cid,
			AID:     aid,
			UID:     uid,
			Path:    syspath,
			Latency: int(latency / time.Millisecond),
		})
	}

	WriteOK(w, r, &ret)
}

// The End.
