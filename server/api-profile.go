package hms

import (
	"encoding/xml"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"github.com/studio-b12/gowebdav"
	"golang.org/x/crypto/ssh"
)

// Add new drive location for profile.
func SpiDriveAdd(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path" binding:"required"`
		Name string `json:"name" yaml:"name" xml:"name"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Added bool    `json:"added" yaml:"added" xml:"added"`
		FP    FileKit `json:"fp" yaml:"fp" xml:"fp"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_drvadd_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_drvadd_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_drvadd_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_drvadd_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var fpath = ToSlash(arg.Path)
	var syspath string
	var puid Puid_t
	var fi fs.FileInfo
	if fi, _ = JP.Stat(fpath); fi != nil {
		syspath = path.Clean(fpath)
		// append slash to disk root to prevent open current dir on this disk
		if syspath[len(syspath)-1] == ':' { // syspath here is always have non zero length
			syspath += "/"
		}
		puid = PathStoreCache(session, syspath)
	} else {
		if syspath, puid, err = UnfoldPath(session, fpath); err != nil {
			Ret400(c, AEC_drvadd_badpath, err)
			return
		}
		if fi, err = JP.Stat(syspath); err != nil {
			Ret404(c, AEC_drvadd_miss, http.ErrMissingFile)
			return
		}
	}

	if Hidden.Fits(syspath) {
		Ret403(c, AEC_drvadd_hidden, ErrHidden)
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

	RetOk(c, ret)
}

// Remove drive from profile with given identifier.
func SpiDriveDel(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_drvdel_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_drvdel_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_drvdel_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_drvdel_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, ok = PathStorePath(session, arg.PUID); !ok {
		Ret404(c, AEC_drvdel_nopath, ErrNoPath)
		return
	}

	ret.Deleted = acc.DelLocal(syspath)

	RetOk(c, ret)
}

// Add new cloud entry for profile.
func SpiCloudAdd(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Scheme   string `json:"scheme" yaml:"scheme" xml:"scheme"`
		Host     string `json:"host" yaml:"host" xml:"host" binding:"required"`
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

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_cldadd_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_cldadd_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_cldadd_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_cldadd_deny, ErrDeny)
		return
	}

	var argurl *url.URL
	if argurl, err = url.Parse(arg.Host); err != nil {
		Ret400(c, AEC_cldadd_badhost, err)
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
		if conn, err = ftp.Dial(argurl.Host, ftp.DialWithTimeout(Cfg.DialTimeout)); err != nil {
			Ret404(c, AEC_cldadd_ftpdial, err)
			return
		}
		defer conn.Quit()
		if err = conn.Login(arg.Login, arg.Password); err != nil {
			Ret403(c, AEC_cldadd_ftpcred, err)
			return
		}

		var root *ftp.Entry
		if root, err = conn.GetEntry(""); err != nil {
			Ret403(c, AEC_cldadd_ftproot, err)
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
			Ret404(c, AEC_cldadd_sftpdial, err)
			return
		}
		defer conn.Close()

		var client *sftp.Client
		if client, err = sftp.NewClient(conn); err != nil {
			Ret403(c, AEC_cldadd_sftpcli, err)
			return
		}
		defer client.Close()

		var pwd string
		if pwd, err = client.Getwd(); err != nil {
			Ret403(c, AEC_cldadd_sftppwd, err)
			return
		}

		var root fs.FileInfo
		if root, err = client.Lstat(path.Join(pwd, "/")); err != nil {
			Ret403(c, AEC_cldadd_sftproot, err)
			return
		}
		size, mtime = int64(root.Size()), root.ModTime()

	case "http", "https":
		var client = gowebdav.NewClient(surl, "", "") // user & password gets from URL
		var fi fs.FileInfo
		if fi, err = client.Stat(""); err != nil || !fi.IsDir() {
			Ret404(c, AEC_cldadd_davdial, err)
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

	RetOk(c, ret)
}

// Remove cloud entry from profile with given identifier.
func SpiCloudDel(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_clddel_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_clddel_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_clddel_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_clddel_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, ok = PathStorePath(session, arg.PUID); !ok {
		Ret404(c, AEC_clddel_nopath, ErrNoPath)
		return
	}

	ret.Deleted = acc.DelCloud(syspath)

	RetOk(c, ret)
}

// APIHANDLER
func SpiShareAdd(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Shared bool `json:"shared" yaml:"shared" xml:"shared"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_shradd_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_shradd_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_shradd_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_shradd_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, _, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		Ret404(c, AEC_shradd_nopath, err)
		return
	}
	if !acc.PathAdmin(syspath) {
		Ret403(c, AEC_shradd_access, ErrNoAccess)
		return
	}

	ret.Shared = acc.AddShare(syspath)
	Log.Infof("id%d: add share '%s'", acc.ID, syspath)

	RetOk(c, ret)
}

// APIHANDLER
func SpiShareDel(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Path string `json:"path" yaml:"path" xml:"path" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Deleted bool `json:"deleted" yaml:"deleted" xml:"deleted"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_shrdel_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = GetAID(c); err != nil {
		Ret400(c, AEC_shrdel_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, AEC_shrdel_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, AEC_shrdel_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var syspath string
	if syspath, _, err = UnfoldPath(session, ToSlash(arg.Path)); err != nil {
		Ret404(c, AEC_shrdel_nopath, ErrNoPath)
		return
	}
	if !acc.PathAdmin(syspath) {
		Ret403(c, AEC_shrdel_access, ErrNoAccess)
		return
	}

	if ret.Deleted = acc.DelShare(syspath); ret.Deleted {
		Log.Infof("id%d: delete share '%s'", acc.ID, syspath)
	}

	RetOk(c, ret)
}

// The End.
