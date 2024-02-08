package hms

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHANDLER
func SpiPage(pref, fname string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var content, ok = pagecache[pref+"/"+fname]
		if !ok {
			Ret404(c, SEC_page_absent, fs.ErrNotExist)
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(c.Writer, c.Request, fname, starttime, bytes.NewReader(content))
	}
}

// Hands out converted media files if them can be cached,
// or file system content as is.
func SpiFile(c *gin.Context) {
	var err error
	var ok bool

	// get arguments
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_media_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_media_noacc, ErrNoAcc)
		return
	}
	var fpath = strings.TrimPrefix(c.Param("path"), "/")

	var media bool
	if s, ok := c.GetQuery("media"); ok {
		if media, err = strconv.ParseBool(s); err != nil {
			Ret400(c, SEC_media_badmedia, ErrArgNoHD)
			return
		}
	}
	var hd bool
	if s, ok := c.GetQuery("hd"); ok {
		if hd, err = strconv.ParseBool(s); err != nil {
			Ret400(c, SEC_media_badhd, ErrArgNoHD)
			return
		}
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	var puid Puid_t
	if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
		Ret400(c, SEC_media_badpath, err)
		return
	}

	if Hidden.Fits(syspath) {
		Ret403(c, SEC_media_hidden, ErrHidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		Ret403(c, SEC_media_access, ErrNoAccess)
		return
	}

	var grp = GetFileGroup(syspath)
	if hd && grp == FGimage {
		var md MediaData
		if md, err = HdCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				RetErr(c, http.StatusGone, SEC_media_hdgone, err)
				return
			}
			if !errors.Is(err, ErrNotHD) {
				Ret500(c, SEC_media_hdfail, err)
				return
			}
		} else {
			if md.Mime == MimeNil {
				Ret500(c, SEC_media_hdnocnt, ErrBadMedia)
				return
			}

			if HasRangeBegin(c.Request) { // beginning of content
				Log.Infof("id%d: media-hd %s", acc.ID, path.Base(syspath))
				go XormUserlog.InsertOne(&OpenStore{
					UAID:    RequestUAID(c.Request),
					AID:     aid,
					UID:     uid,
					Path:    syspath,
					Latency: -1,
				})
			}
			c.Header("Content-Type", MimeStr[md.Mime])
			http.ServeContent(c.Writer, c.Request, syspath, md.Time, bytes.NewReader(md.Data))
			return
		}
	}

	if media && grp == FGimage {
		var md MediaData
		if md, err = MediaCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				RetErr(c, http.StatusGone, SEC_media_medgone, err)
				return
			}
			if !errors.Is(err, ErrUncacheable) {
				Ret404(c, SEC_media_medfail, err)
				return
			}
		} else {
			if md.Mime == MimeNil {
				Ret500(c, SEC_media_mednocnt, ErrBadMedia)
				return
			}

			if HasRangeBegin(c.Request) { // beginning of content
				Log.Infof("id%d: media %s", acc.ID, path.Base(syspath))
				go XormUserlog.InsertOne(&OpenStore{
					UAID:    RequestUAID(c.Request),
					AID:     aid,
					UID:     uid,
					Path:    syspath,
					Latency: -1,
				})
			}
			c.Header("Content-Type", MimeStr[md.Mime])
			http.ServeContent(c.Writer, c.Request, syspath, md.Time, bytes.NewReader(md.Data))
			return
		}
	}

	if HasRangeBegin(c.Request) { // beginning of content
		Log.Infof("id%d: serve %s", acc.ID, path.Base(syspath))
		go XormUserlog.InsertOne(&OpenStore{
			UAID:    RequestUAID(c.Request),
			AID:     aid,
			UID:     uid,
			Path:    syspath,
			Latency: -1,
		})
	}

	var content RFile
	if content, err = OpenFile(syspath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// try to redirect to external shared file (not at DAV-disk)
			if IsRemote(syspath) {
				http.Redirect(c.Writer, c.Request, syspath, http.StatusMovedPermanently)
				return
			}
		}
		if errors.Is(err, fs.ErrNotExist) {
			RetErr(c, http.StatusGone, SEC_media_filegone, err)
			return
		}
		Ret500(c, SEC_media_fileopen, err)
		return
	}
	defer content.Close()

	var t time.Time
	if IsRemote(syspath) {
		t = starttime
	} else {
		if fi, _ := content.Stat(); fi != nil {
			t = fi.ModTime()
		}
	}

	http.ServeContent(c.Writer, c.Request, syspath, t, content)
}

// Hands out embedded thumbnails for given files if any.
func SpiEtmb(c *gin.Context) {
	var err error
	var ok bool

	// get arguments
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_etmb_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_etmb_noacc, ErrNoAcc)
		return
	}
	var puid Puid_t
	if err = puid.Set(c.Param("puid")); err != nil {
		Ret400(c, SEC_etmb_nopuid, err)
		return
	}

	var syspath string
	if syspath, ok = PathCache.GetDir(puid); !ok {
		Ret404(c, SEC_etmb_nopath, ErrNoPath)
		return
	}

	if Hidden.Fits(syspath) {
		Ret403(c, SEC_etmb_hidden, ErrHidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		Ret403(c, SEC_etmb_access, ErrNoAccess)
		return
	}

	var md MediaData
	if md, err = ExtractThmub(syspath); err != nil {
		if errors.Is(err, ErrNoThumb) {
			RetErr(c, http.StatusNoContent, SEC_etmb_notmb, err)
			return
		} else {
			Ret500(c, SEC_etmb_badcnt, err)
			return
		}
	}
	c.Header("Content-Type", MimeStr[md.Mime])
	http.ServeContent(c.Writer, c.Request, syspath, md.Time, bytes.NewReader(md.Data))
}

// Hands out cached thumbnails for given files.
func SpiMtmb(c *gin.Context) {
	var err error
	var ok bool

	// get arguments
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_mtmb_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_mtmb_noacc, ErrNoAcc)
		return
	}
	var puid Puid_t
	if err = puid.Set(c.Param("puid")); err != nil {
		Ret400(c, SEC_mtmb_nopuid, err)
		return
	}

	var syspath string
	if syspath, ok = PathCache.GetDir(puid); !ok {
		Ret404(c, SEC_mtmb_nopath, ErrNoPath)
		return
	}

	if Hidden.Fits(syspath) {
		Ret403(c, SEC_mtmb_hidden, ErrHidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		Ret403(c, SEC_mtmb_access, ErrNoAccess)
		return
	}

	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = ThumbPkg.GetFile(syspath); err != nil {
		Ret500(c, SEC_mtmb_badcnt, err)
		return
	}
	if file == nil {
		Ret404(c, SEC_mtmb_absent, fs.ErrNotExist)
		return
	}
	defer file.Close()

	c.Header("Content-Type", mime)
	http.ServeContent(c.Writer, c.Request, syspath, t, file)
}

// Hands out thumbnails for given files if them cached.
func SpiTile(c *gin.Context) {
	var err error
	var ok bool

	// get arguments
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_tile_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_tile_noacc, ErrNoAcc)
		return
	}
	var puid Puid_t
	if err = puid.Set(c.Param("puid")); err != nil {
		Ret400(c, SEC_tile_nopuid, err)
		return
	}
	var dim = strings.Split(c.Param("dim"), "x")
	if len(dim) != 2 {
		Ret400(c, SEC_tile_twodim, ErrArgNoDim)
		return
	}
	var wdh, hgt int
	if wdh, err = strconv.Atoi(dim[0]); err != nil {
		Ret400(c, SEC_tile_badwdh, ErrArgNoDim)
		return
	}
	if hgt, err = strconv.Atoi(dim[1]); err != nil {
		Ret400(c, SEC_tile_badhgt, ErrArgNoDim)
		return
	}
	if wdh == 0 || hgt == 0 {
		Ret400(c, SEC_tile_zero, ErrArgZDim)
		return
	}

	var syspath string
	if syspath, ok = PathCache.GetDir(puid); !ok {
		Ret404(c, SEC_tile_nopath, ErrNoPath)
		return
	}

	if Hidden.Fits(syspath) {
		Ret403(c, SEC_tile_hidden, ErrHidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		Ret403(c, SEC_tile_access, ErrNoAccess)
		return
	}

	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)
	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = TilesPkg.GetFile(tilepath); err != nil {
		Ret500(c, SEC_tile_badcnt, err)
		return
	}
	if file == nil {
		Ret404(c, SEC_tile_absent, fs.ErrNotExist)
		return
	}
	defer file.Close()

	c.Header("Content-Type", mime)
	http.ServeContent(c.Writer, c.Request, syspath, t, file)
}

// The End.
