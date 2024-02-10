package hms

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
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
func SpiFolder(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path,omitempty" yaml:"path,omitempty" xml:"path,omitempty,attr" binding:"required"`
		Scan bool   `json:"scan,omitempty" yaml:"scan,omitempty" xml:"scan,omitempty"`
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

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_folder_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_folder_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_folder_noacc, ErrNoAcc)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	var puid Puid_t
	if syspath, puid, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		Ret400(c, SEC_folder_badpath, err)
		return
	}

	if Hidden.Fits(syspath) {
		Ret403(c, SEC_folder_hidden, ErrHidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		Ret403(c, SEC_folder_access, ErrNoAccess)
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
	var ip = net.ParseIP(c.RemoteIP())
	ret.Access = InPasslist(ip)

	var t = time.Now()
	if puid < PUIDcache {
		if uid != aid && !acc.IsShared(syspath) {
			Ret403(c, SEC_folder_noshr, ErrNotShared)
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

			var dp DirProp
			if ret.List, dp, err = ScanFileNameList(acc, session, vfiles, arg.Scan); err != nil {
				Ret500(c, SEC_folder_home, err)
				return
			}

			go SqlSession(func(session *Session) (res any, err error) {
				DirStoreSet(session, puid, dp)
				return
			})
		case PUIDlocal:
			if ret.List, err = acc.ScanLocal(session, arg.Scan); err != nil {
				Ret500(c, SEC_folder_drives, err)
				return
			}
		case PUIDremote:
			if ret.List, err = acc.ScanRemote(session, arg.Scan); err != nil {
				Ret500(c, SEC_folder_remote, err)
				return
			}
		case PUIDshares:
			if ret.List, err = acc.ScanShares(session, arg.Scan); err != nil {
				Ret500(c, SEC_folder_shares, err)
				return
			}
		case PUIDmedia, PUIDvideo, PUIDaudio, PUIDimage, PUIDbooks, PUIDtexts:
			if ret.List, err = ScanCat(acc, session, puid, catcolumn[puid], 0.5, arg.Scan); err != nil {
				Ret500(c, SEC_folder_media, err)
				return
			}
		case PUIDmap:
			var n = 0
			var vfiles []fs.FileInfo // verified file infos
			var vpaths []DiskPath    // verified paths
			GpsCache.Range(func(puid Puid_t, gps GpsInfo) bool {
				if fpath, ok := PathStorePath(session, puid); ok {
					if !Hidden.Fits(fpath) && acc.PathAccess(fpath, uid == aid) {
						if fi, _ := JP.Stat(fpath); fi != nil {
							vfiles = append(vfiles, fi)
							vpaths = append(vpaths, MakeFilePath(fpath))
							n++
						}
					}
				}
				return Cfg.RangeSearchAny <= 0 || n < Cfg.RangeSearchAny
			})
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths, arg.Scan); err != nil {
				Ret500(c, SEC_folder_map, err)
				return
			}
		default:
			Ret404(c, SEC_folder_nocat, ErrNoCat)
			return
		}
		ret.Static = true
	} else {
		var fi fs.FileInfo
		if fi, err = JP.Stat(syspath); err != nil {
			Ret500(c, SEC_folder_stat, err)
			return
		}
		ret.Static = IsStatic(fi) || !fi.IsDir()

		var ext = arg.Ext
		if ext == "" && !fi.IsDir() {
			ext = GetFileExt(syspath)
		}

		if fi.IsDir() || IsTypeISO(ext) {
			if ret.List, ret.Skipped, err = ScanDir(acc, session, syspath, uid == aid, arg.Scan); err != nil && len(ret.List) == 0 {
				if errors.Is(err, fs.ErrNotExist) {
					Ret404(c, SEC_folder_absent, err)
				} else {
					Ret500(c, SEC_folder_fail, err)
				}
				return
			}
		} else if IsTypePlaylist(ext) {
			var file fs.File
			if file, err = JP.Open(syspath); err != nil {
				Ret500(c, SEC_folder_open, err)
				return
			}
			defer file.Close()

			var pl Playlist
			pl.Dest = path.Dir(syspath)
			switch ext {
			case ".m3u", ".m3u8":
				if _, err = pl.ReadM3U(file); err != nil {
					RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_m3u, err)
					return
				}
			case ".wpl":
				if _, err = pl.ReadWPL(file); err != nil {
					RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_wpl, err)
					return
				}
			case ".pls":
				if _, err = pl.ReadPLS(file); err != nil {
					RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_pls, err)
					return
				}
			case ".asx":
				if _, err = pl.ReadASX(file); err != nil {
					RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_asx, err)
					return
				}
			case ".xspf":
				if _, err = pl.ReadXSPF(file); err != nil {
					RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_xspf, err)
					return
				}
			default:
				RetErr(c, http.StatusUnsupportedMediaType, SEC_folder_format, ErrNotPlay)
				return
			}

			var vfiles []fs.FileInfo // verified file infos
			var vpaths []DiskPath    // verified paths
			for _, track := range pl.Tracks {
				var fpath = ToSlash(track.Location)
				if !Hidden.Fits(fpath) && acc.PathAccess(fpath, uid == aid) {
					if fi, _ := JP.Stat(fpath); fi != nil {
						vfiles = append(vfiles, fi)
						vpaths = append(vpaths, MakeFilePath(fpath))
					}
				}
			}
			if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths, arg.Scan); err != nil {
				Ret500(c, SEC_folder_tracks, err)
				return
			}
			ret.Skipped = len(pl.Tracks) - len(ret.List)
		}
	}

	var latency = time.Since(t)
	Log.Infof("id%d: navigate to %s, items %d, timeout %s", acc.ID, syspath, len(ret.List), latency)
	go XormUserlog.InsertOne(&OpenStore{
		UAID:    RequestUAID(c),
		AID:     aid,
		UID:     uid,
		Path:    syspath,
		Latency: int(latency / time.Millisecond),
	})

	RetOk(c, ret)
}

// The End.
