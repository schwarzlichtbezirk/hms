package hms

import (
	"encoding/xml"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"github.com/studio-b12/gowebdav"
	"golang.org/x/crypto/ssh"
)

// APIHANDLER
func drvaddAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path"`
		Name string `json:"name" yaml:"name" xml:"name"`
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
	if acc = ProfileByID(aid); acc == nil {
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

	var session = XormStorage.NewSession()
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

	var name = arg.Name
	if len(name) == 0 {
		if strings.HasSuffix(name, ":/") {
			name = "disk " + strings.ToUpper(name[0:1])
		} else {
			name = path.Base(syspath)
		}
	}

	var fk FileKit
	fk.PUID = puid
	fk.Free = acc.PathAccess(syspath, false)
	fk.Shared = acc.IsShared(syspath)
	fk.Static = IsStatic(fi)
	fk.Name = name
	fk.Type = FTdrv
	fk.Size = fi.Size()
	fk.Time = fi.ModTime()

	ret.FP = fk
	ret.Added = acc.AddLocal(syspath, name)

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
	if acc = ProfileByID(aid); acc == nil {
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

	var session = XormStorage.NewSession()
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

		Scheme   string `json:"scheme" yaml:"scheme" xml:"scheme"`
		Host     string `json:"host" yaml:"host" xml:"host"`
		Port     string `json:"port" yaml:"port" xml:"port"`
		Login    string `json:"login,omitempty" yaml:"login,omitempty" xml:"login,omitempty"`
		Password string `json:"password,omitempty" yaml:"password,omitempty" xml:"password,omitempty"`
		Name     string `json:"name" yaml:"name" xml:"name"`
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
	if acc = ProfileByID(aid); acc == nil {
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

	var argurl *url.URL
	if argurl, err = url.Parse(arg.Host); err != nil {
		WriteError400(w, r, err, AECcldaddbadhost)
		return
	}
	if argurl.Scheme == "" {
		argurl.Scheme = arg.Scheme
	}
	if i := strings.Index(argurl.Host, ":"); i == -1 && arg.Port != "" {
		argurl.Host += ":" + arg.Port
	}
	if argurl.User.String() == "" {
		argurl.User = url.UserPassword(arg.Login, arg.Password)
	}
	var surl = argurl.String()

	var name = arg.Name
	if len(name) == 0 {
		name = argurl.Redacted()
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var (
		size  int64
		mtime time.Time
	)
	switch arg.Scheme {
	case "ftp":
		var conn *ftp.ServerConn
		if conn, err = ftp.Dial(argurl.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddftpdial)
			return
		}
		defer conn.Quit()
		if err = conn.Login(arg.Login, arg.Password); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddftpcred)
			return
		}

		var root *ftp.Entry
		if root, err = conn.GetEntry(""); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddftproot)
			return
		}
		size, mtime = int64(root.Size), root.Time

	case "sftp":
		var conn *ssh.Client
		var config = &ssh.ClientConfig{
			User: arg.Login,
			Auth: []ssh.AuthMethod{
				ssh.Password(arg.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		if conn, err = ssh.Dial("tcp", argurl.Host, config); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddsftpdial)
			return
		}
		defer conn.Close()

		var client *sftp.Client
		if client, err = sftp.NewClient(conn); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddsftpcli)
			return
		}
		defer client.Close()

		var pwd string
		if pwd, err = client.Getwd(); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddsftppwd)
			return
		}

		var root fs.FileInfo
		if root, err = client.Lstat(path.Join(pwd, "/")); err != nil {
			WriteError(w, r, http.StatusNotFound, err, AECcldaddsftproot)
			return
		}
		size, mtime = int64(root.Size()), root.ModTime()

	case "http", "https":
		var client = gowebdav.NewClient(surl, "", "") // user & password gets from URL
		var fi fs.FileInfo
		if fi, err = client.Stat(""); err != nil || !fi.IsDir() {
			WriteError(w, r, http.StatusNotFound, err, AECcldadddavdial)
			return
		}
		size, mtime = 0, time.Unix(0, 0) // DAV does not provides info for folders
	}

	var fk FileKit
	fk.PUID = PathStoreCache(session, surl)
	fk.Free = acc.PathAccess(surl, false)
	fk.Shared = acc.IsShared(surl)
	fk.Static = false
	fk.Name = name
	fk.Type = FTcld
	fk.Size = size
	fk.Time = mtime

	ret.FP = fk
	ret.Added = acc.AddCloud(surl, name)

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
	if acc = ProfileByID(aid); acc == nil {
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

	var session = XormStorage.NewSession()
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
	if acc = ProfileByID(aid); acc == nil {
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

	var session = XormStorage.NewSession()
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
	if acc = ProfileByID(aid); acc == nil {
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

	var session = XormStorage.NewSession()
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
