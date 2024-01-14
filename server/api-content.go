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

	"github.com/gorilla/mux"
)

// APIHANDLER
func pageHandler(pref, fname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var content, ok = pagecache[pref+"/"+fname]
		if !ok {
			WriteError(w, r, http.StatusNotFound, fs.ErrNotExist, SEC_page_absent)
			return
		}

		WriteHTMLHeader(w)
		http.ServeContent(w, r, fname, starttime, bytes.NewReader(content))
	}
}

// Hands out converted media files if them can be cached.
func fileHandler(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, SEC_media_noacc)
		return
	}

	// get arguments
	var media bool
	if s := r.FormValue("media"); len(s) > 0 {
		if media, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, SEC_media_badmedia)
			return
		}
	}
	var hd bool
	if s := r.FormValue("hd"); len(s) > 0 {
		if hd, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, SEC_media_badhd)
			return
		}
	}

	var chunks = strings.Split(r.URL.Path, "/")
	if len(chunks) < 4 {
		panic("bad route for URL " + r.URL.Path)
	}
	var fpath = strings.Join(chunks[3:], "/")

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	var puid Puid_t
	if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
		WriteError400(w, r, err, SEC_media_badpath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, SEC_media_hidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, SEC_media_access)
		return
	}

	var grp = GetFileGroup(syspath)
	if hd && grp == FGimage {
		var md MediaData
		if md, err = HdCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, SEC_media_hdgone)
				return
			}
			if !errors.Is(err, ErrNotHD) {
				WriteError500(w, r, err, SEC_media_hdfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, SEC_media_hdnocnt)
				return
			}

			if HasRangeBegin(r) { // beginning of content
				Log.Infof("id%d: media-hd %s", acc.ID, path.Base(syspath))
				go XormUserlog.InsertOne(&OpenStore{
					UAID:    RequestUAID(r),
					AID:     aid,
					UID:     uid,
					Path:    syspath,
					Latency: -1,
				})
			}
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, syspath, md.Time, bytes.NewReader(md.Data))
			return
		}
	}

	if media && grp == FGimage {
		var md MediaData
		if md, err = MediaCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, SEC_media_medgone)
				return
			}
			if !errors.Is(err, ErrUncacheable) {
				WriteError(w, r, http.StatusNotFound, err, SEC_media_medfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, SEC_media_mednocnt)
				return
			}

			if HasRangeBegin(r) { // beginning of content
				Log.Infof("id%d: media %s", acc.ID, path.Base(syspath))
				go XormUserlog.InsertOne(&OpenStore{
					UAID:    RequestUAID(r),
					AID:     aid,
					UID:     uid,
					Path:    syspath,
					Latency: -1,
				})
			}
			w.Header().Set("Content-Type", MimeStr[md.Mime])
			http.ServeContent(w, r, syspath, md.Time, bytes.NewReader(md.Data))
			return
		}
	}

	if HasRangeBegin(r) { // beginning of content
		Log.Infof("id%d: serve %s", acc.ID, path.Base(syspath))
		go XormUserlog.InsertOne(&OpenStore{
			UAID:    RequestUAID(r),
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
				http.Redirect(w, r, syspath, http.StatusMovedPermanently)
				return
			}
		}
		if errors.Is(err, fs.ErrNotExist) {
			WriteError(w, r, http.StatusGone, err, SEC_media_filegone)
			return
		}
		WriteError500(w, r, err, SEC_media_fileopen)
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

	WriteStdHeader(w)
	http.ServeContent(w, r, syspath, t, content)
}

// Hands out embedded thumbnails for given files if any.
func etmbHandler(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, SEC_etmb_noacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, SEC_etmb_nopuid)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, SEC_etmb_nopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, SEC_etmb_hidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, SEC_etmb_access)
		return
	}

	var md MediaData
	if md, err = ExtractThmub(session, syspath); err != nil {
		if errors.Is(err, ErrNoThumb) {
			WriteError(w, r, http.StatusNoContent, err, SEC_etmb_notmb)
			return
		} else {
			WriteError500(w, r, err, SEC_etmb_badcnt)
			return
		}
	}
	w.Header().Set("Content-Type", MimeStr[md.Mime])
	http.ServeContent(w, r, syspath, md.Time, bytes.NewReader(md.Data))
}

// Hands out cached thumbnails for given files.
func mtmbHandler(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, SEC_mtmb_noacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, SEC_mtmb_nopuid)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, SEC_mtmb_nopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, SEC_mtmb_hidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, SEC_mtmb_access)
		return
	}

	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = ThumbPkg.GetFile(syspath); err != nil {
		WriteError500(w, r, err, SEC_mtmb_badcnt)
		return
	}
	if file == nil {
		WriteError(w, r, http.StatusNotFound, fs.ErrNotExist, SEC_mtmb_absent)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", mime)
	http.ServeContent(w, r, syspath, t, file)
}

// Hands out thumbnails for given files if them cached.
func tileHandler(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, SEC_tile_noacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, SEC_tile_nopuid)
		return
	}
	var wdh, _ = strconv.Atoi(vars["wdh"])
	var hgt, _ = strconv.Atoi(vars["hgt"])
	if wdh == 0 || hgt == 0 {
		WriteError400(w, r, ErrArgNoDim, SEC_tile_baddim)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, SEC_tile_nopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, SEC_tile_hidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, SEC_tile_access)
		return
	}

	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)
	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = TilesPkg.GetFile(tilepath); err != nil {
		WriteError500(w, r, err, SEC_tile_badcnt)
		return
	}
	if file == nil {
		WriteError(w, r, http.StatusNotFound, fs.ErrNotExist, SEC_tile_absent)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", mime)
	http.ServeContent(w, r, syspath, t, file)
}

// The End.
