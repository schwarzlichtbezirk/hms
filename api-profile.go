package hms

import (
	"encoding/xml"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/jlaffaye/ftp"
)

// APIHANDLER
func drvaddAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Added bool    `json:"added" yaml:"added" xml:"added"`
		FP    FileKit `json:"fp" yaml:"fp" xml:"fp"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvaddnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvadddeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvaddnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var fpath = ToSlash(arg.Path)
	var syspath string
	var puid Puid_t
	var fi fs.FileInfo
	if fi, _ = StatFile(fpath); fi != nil {
		syspath = path.Clean(fpath)
		// append slash to disk root to prevent open current dir on this disk
		if syspath[len(syspath)-1] == ':' { // syspath here is always have non zero length
			syspath += "/"
		}
		puid = PathStoreCache(session, syspath)
	} else {
		if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
			WriteError400(w, r, err, AECdrvaddbadpath)
			return
		}
		if fi, err = StatFile(syspath); err != nil {
			WriteError(w, r, http.StatusNotFound, http.ErrMissingFile, AECdrvaddmiss)
			return
		}
	}

	if acc.IsHidden(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrHidden, AECdrvaddhidden)
		return
	}

	var fk FileKit
	fk.PUID = puid
	fk.Free = acc.PathAccess(syspath, false)
	fk.Shared = acc.IsShared(syspath)
	if fi != nil {
		_, fk.Static = fi.(*FileInfoISO)
	} else {
		fk.Static = true
	}
	fk.Name = path.Base(syspath)
	fk.Type = FTdrv
	fk.Size = fi.Size()
	fk.Time = fi.ModTime()

	ret.FP = fk
	ret.Added = acc.AddLocal(syspath)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func drvdelAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECdrvdelnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvdeldeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECdrvdelnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECdrvdelnopath)
	}

	ret.Deleted = acc.DelLocal(syspath)
	WriteOK(w, r, &ret)
}

// APIHANDLER
func cldaddAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Host string `json:"host" yaml:"host" xml:"host"`
		Port int    `json:"port" yaml:"port" xml:"port"`
		Name string `json:"name,omitempty" yaml:"name,omitempty" xml:"name,omitempty"`
		Pass string `json:"pass,omitempty" yaml:"pass,omitempty" xml:"pass,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Added bool    `json:"added" yaml:"added" xml:"added"`
		FP    FileKit `json:"fp" yaml:"fp" xml:"fp"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECcldaddnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECdrvadddeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Host) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECcldaddnodata)
		return
	}
	if arg.Port == 0 {
		arg.Port = 21
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var host = arg.Host
	if arg.Port > 0 {
		host += ":" + strconv.Itoa(arg.Port)
	}
	var u = url.URL{
		Scheme: "ftp",
		User:   url.UserPassword(arg.Name, arg.Pass),
		Host:   host,
	}
	var syspath = u.String()

	var conn *ftp.ServerConn
	if conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(5*time.Second)); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECcldaddnodial)
		return
	}
	defer conn.Quit()

	if err = conn.Login(arg.Name, arg.Pass); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECcldaddcred)
		return
	}

	var root *ftp.Entry
	if root, err = conn.GetEntry(""); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECcldaddroot)
		return
	}

	var fk FileKit
	fk.PUID = PathStoreCache(session, syspath)
	fk.Free = acc.PathAccess(syspath, false)
	fk.Shared = acc.IsShared(syspath)
	fk.Static = false
	fk.Name = path.Base(syspath)
	fk.Type = FTcld
	fk.Size = int64(root.Size)
	fk.Time = root.Time

	ret.FP = fk
	ret.Added = acc.AddCloud(syspath)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func clddelAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECclddelnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECclddeldeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.PUID == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECclddelnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var syspath, ok = PathStorePath(session, arg.PUID)
	if !ok {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECclddelnopath)
	}

	ret.Deleted = acc.DelCloud(syspath)
	WriteOK(w, r, &ret)
}

// APIHANDLER
func shraddAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Shared bool `json:"shared" yaml:"shared" xml:"shared"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECshraddnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECshradddeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshraddnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, _, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECshraddnopath)
	}
	if !acc.PathAdmin(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECshraddaccess)
		return
	}

	ret.Shared = acc.AddShare(syspath)
	Log.Infof("id%d: add share '%s'", acc.ID, syspath)

	WriteOK(w, r, &ret)
}

// APIHANDLER
func shrdelAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECshrdelnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECshrdeldeny)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Path) == 0 {
		WriteError400(w, r, ErrArgNoPuid, AECshrdelnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, _, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		WriteError(w, r, http.StatusNotFound, ErrNoPath, AECshrdelnopath)
	}
	if !acc.PathAdmin(syspath) {
		WriteError(w, r, http.StatusForbidden, ErrNoAccess, AECshrdelaccess)
		return
	}

	if ret.Deleted = acc.DelShare(syspath); ret.Deleted {
		Log.Infof("id%d: delete share '%s'", acc.ID, syspath)
	}

	WriteOK(w, r, &ret)
}

// The End.
