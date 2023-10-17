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

	. "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/gorilla/mux"
)

// APIHANDLER
func pageHandler(pref, fname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var content, ok = pagecache[pref+"/"+fname]
		if !ok {
			WriteError(w, r, http.StatusNotFound, ErrNotFound, AECpageabsent)
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
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECmedianoacc)
		return
	}

	// get arguments
	var media bool
	if s := r.FormValue("media"); len(s) > 0 {
		if media, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, AECmediabadmedia)
			return
		}
	}
	var hd bool
	if s := r.FormValue("hd"); len(s) > 0 {
		if hd, err = strconv.ParseBool(s); err != nil {
			WriteError400(w, r, ErrArgNoHD, AECmediabadhd)
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
		WriteError400(w, r, err, AECmediabadpath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECmediahidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmediaaccess)
		return
	}

	var grp = GetFileGroup(syspath)
	if hd && grp == FGimage {
		var md MediaData
		if md, err = HdCacheGet(session, puid); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				WriteError(w, r, http.StatusGone, err, AECmediahdgone)
				return
			}
			if !errors.Is(err, ErrNotHD) {
				WriteError500(w, r, err, AECmediahdfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, AECmediahdnocnt)
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
				WriteError(w, r, http.StatusGone, err, AECmediamedgone)
				return
			}
			if !errors.Is(err, ErrUncacheable) {
				WriteError(w, r, http.StatusNotFound, err, AECmediamedfail)
				return
			}
		} else {
			if md.Mime == MimeNil {
				WriteError500(w, r, ErrBadMedia, AECmediamednocnt)
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

	var content File
	if content, err = OpenFile(syspath); err != nil {
		if errors.Is(err, ErrNotFound) {
			// try to redirect to external shared file (not at DAV-disk)
			if IsRemote(syspath) {
				http.Redirect(w, r, syspath, http.StatusMovedPermanently)
				return
			}
		}
		if errors.Is(err, fs.ErrNotExist) {
			WriteError(w, r, http.StatusGone, err, AECmediafilegone)
			return
		}
		WriteError500(w, r, err, AECmediafileopen)
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
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECetmbnoacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECetmbnopuid)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECetmbnopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECetmbhidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECetmbaccess)
		return
	}

	var md MediaData
	if md, err = ExtractThmub(session, syspath); err != nil {
		if errors.Is(err, ErrNoThumb) {
			WriteError(w, r, http.StatusNoContent, err, AECetmbnotmb)
			return
		} else {
			WriteError500(w, r, err, AECetmbbadcnt)
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
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECmtmbnoacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECmtmbnopuid)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECmtmbnopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECmtmbhidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECmtmbaccess)
		return
	}

	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = ThumbPkg.GetFile(syspath); err != nil {
		WriteError500(w, r, err, AECmtmbbadcnt)
		return
	}
	if file == nil {
		WriteError(w, r, http.StatusNotFound, ErrNotFound, AECmtmbabsent)
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
		WriteError(w, r, http.StatusNotFound, ErrNoAcc, AECtilenoacc)
		return
	}

	// get arguments
	var vars = mux.Vars(r)
	var puid Puid_t
	if err = puid.Set(vars["puid"]); err != nil {
		WriteError400(w, r, err, AECtilenopuid)
		return
	}
	var wdh, _ = strconv.Atoi(vars["wdh"])
	var hgt, _ = strconv.Atoi(vars["hgt"])
	if wdh == 0 || hgt == 0 {
		WriteError400(w, r, ErrArgNoDim, AECtilebaddim)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, puid)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECtilenopath)
		return
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECtilehidden)
		return
	}
	if !acc.PathAccess(syspath, uid == aid) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECtileaccess)
		return
	}

	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)
	var file io.ReadSeekCloser
	var mime string
	var t time.Time
	if file, mime, t, err = TilesPkg.GetFile(tilepath); err != nil {
		WriteError500(w, r, err, AECtilebadcnt)
		return
	}
	if file == nil {
		WriteError(w, r, http.StatusNotFound, ErrNotFound, AECtileabsent)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", mime)
	http.ServeContent(w, r, syspath, t, file)
}

// The End.
