package hms

import (
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"time"

	. "github.com/schwarzlichtbezirk/hms/config"
	. "github.com/schwarzlichtbezirk/hms/joint"
)

var catcolumn = map[Puid_t]string{
	PUIDmedia: "video+audio+image",
	PUIDvideo: "video",
	PUIDaudio: "audio",
	PUIDimage: "image",
	PUIDbooks: "books",
	PUIDtexts: "texts",
}

// APIHANDLER
func folderAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path,omitempty" yaml:"path,omitempty" xml:"path,omitempty,attr"`
		Ext  string `json:"ext,omitempty" yaml:"ext,omitempty" xml:"ext,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List    []any  `json:"list" yaml:"list" xml:"list>prop"`
		Skipped int    `json:"skipped" yaml:"skipped" xml:"skipped,attr"`
		PUID    Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`

		SharePath string `json:"sharepath" yaml:"sharepath" xml:"sharepath,attr"`
		ShareName string `json:"sharename" yaml:"sharename" xml:"sharename,attr"`
		RootPath  string `json:"rootpath" yaml:"rootpath" xml:"rootpath,attr"`
		RootName  string `json:"rootname" yaml:"rootname" xml:"rootname,attr"`
		Static    bool   `json:"static" yaml:"static" xml:"static,attr"`

		HasHome bool `json:"hashome" yaml:"hashome" xml:"hashome,attr"`
		Access  bool `json:"access" yaml:"access" xml:"access,attr"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECfoldernoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECfoldernodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

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
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECfolderaccess)
		return
	}

	ret.PUID = puid
	ret.SharePath, ret.ShareName = acc.GetSharePath(session, syspath)
	ret.RootPath, ret.RootName = acc.GetRootPath(session, syspath)

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
	var ip = net.ParseIP(StripPort(r.RemoteAddr))
	ret.Access = InPasslist(ip)

	var t = time.Now()
	if puid < PUIDcache {
		if uid != aid && !acc.IsShared(syspath) {
			WriteError(w, r, http.StatusForbidden, ErrNotShared, AECfoldernoshr)
			return
		}
		switch puid {
		case PUIDhome:
			var vfiles []DiskPath
			for puid := Puid_t(1); puid < PUIDcache; puid++ {
				if puid == PUIDhome {
					continue
				}
				if fpath, ok := CatKeyPath[puid]; ok {
					if uid == aid || acc.IsShared(fpath) {
						vfiles = append(vfiles, DiskPath{fpath, CatNames[fpath]})
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
		case PUIDlocal:
			if ret.List, err = acc.ScanLocal(session); err != nil {
				WriteError500(w, r, err, AECfolderdrives)
				return
			}
		case PUIDremote:
			if ret.List, err = acc.ScanRemote(session); err != nil {
				WriteError500(w, r, err, AECfolderremote)
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
			var vpaths []DiskPath    // verified paths
			gpscache.Enum(func(puid Puid_t, gps GpsInfo) bool {
				if fpath, ok := pathcache.GetDir(puid); ok {
					if !acc.IsHidden(fpath) && acc.PathAccess(fpath, uid == aid) {
						if fi, _ := StatFile(fpath); fi != nil {
							vfiles = append(vfiles, fi)
							vpaths = append(vpaths, MakeFilePath(fpath))
							n++
						}
					}
				}
				return Cfg.RangeSearchAny <= 0 || n < Cfg.RangeSearchAny
			})
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths); err != nil {
				WriteError500(w, r, err, AECfoldermap)
				return
			}
		default:
			WriteError(w, r, http.StatusNotFound, ErrNoCat, AECfoldernocat)
			return
		}
		ret.Static = true
	} else {
		var fi fs.FileInfo
		if fi, err = StatFile(syspath); err != nil {
			WriteError500(w, r, err, AECfolderstat)
			return
		}
		ret.Static = IsStatic(fi) || !fi.IsDir()

		var ext = arg.Ext
		if ext == "" && !fi.IsDir() {
			ext = GetFileExt(syspath)
		}

		if fi.IsDir() || IsTypeISO(ext) {
			if ret.List, ret.Skipped, err = ScanDir(acc, session, syspath, uid == aid); err != nil && len(ret.List) == 0 {
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
			var vpaths []DiskPath    // verified paths
			for _, track := range pl.Tracks {
				var fpath = ToSlash(track.Location)
				if !acc.IsHidden(fpath) && acc.PathAccess(fpath, uid == aid) {
					if fi, _ := StatFile(fpath); fi != nil {
						vfiles = append(vfiles, fi)
						vpaths = append(vpaths, MakeFilePath(fpath))
					}
				}
			}
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths); err != nil {
				WriteError500(w, r, err, AECfoldertracks)
				return
			}
			ret.Skipped = len(pl.Tracks) - len(ret.List)
		}
	}

	var latency = time.Since(t)
	Log.Infof("id%d: navigate to %s, items %d, timeout %s", acc.ID, syspath, len(ret.List), latency)
	go XormUserlog.InsertOne(&OpenStore{
		UAID:    RequestUAID(r),
		AID:     aid,
		UID:     uid,
		Path:    syspath,
		Latency: int(latency / time.Millisecond),
	})

	WriteOK(w, r, &ret)
}

// The End.
