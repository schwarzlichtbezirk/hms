package joint

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"

	"github.com/jlaffaye/ftp"
)

var (
	ErrFtpWhence = errors.New("invalid whence at FTP seeker")
	ErrFtpNegPos = errors.New("negative position at FTP seeker")
)

var (
	pwdmap = map[string]string{}
	pwdmux sync.RWMutex
)

// SplitUrl splits URL to address string and to path as is.
func SplitUrl(urlpath string) (string, string) {
	if i := strings.Index(urlpath, "://"); i != -1 {
		if j := strings.Index(urlpath[i+3:], "/"); j != -1 {
			return urlpath[:i+3+j], urlpath[i+3+j+1:]
		}
		return urlpath, ""
	}
	return "", urlpath
}

// FtpPwd return FTP current directory. It's used cache to avoid
// extra calls to FTP-server to get current directory for every call.
func FtpPwd(ftpaddr string, conn *ftp.ServerConn) (pwd string) {
	pwdmux.RLock()
	pwd, ok := pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if !ok {
		var err error
		if pwd, err = conn.CurrentDir(); err == nil {
			pwd = strings.TrimPrefix(pwd, "/")
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	return
}

// FtpJoint create connection to FTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
type FtpJoint struct {
	key  string // address of FTP-service, i.e. ftp://user:pass@example.com
	conn *ftp.ServerConn
	pwd  string

	path string // path inside of FTP-service
	io.ReadCloser
	pos int64
	end int64
}

func (jnt *FtpJoint) Make(urladdr string) (err error) {
	jnt.key = urladdr
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	if jnt.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.Cfg.DialTimeout)); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	if err = jnt.conn.Login(u.User.Username(), pass); err != nil {
		return
	}
	jnt.pwd = FtpPwd(u.Host, jnt.conn)
	return
}

func (jnt *FtpJoint) Cleanup() error {
	return jnt.conn.Quit()
}

func (jnt *FtpJoint) Key() string {
	return jnt.key
}

func (jnt *FtpJoint) Busy() bool {
	return jnt.path != ""
}

func (jnt *FtpJoint) Open(fpath string) (file fs.File, err error) {
	jnt.path = fpath
	return jnt, nil
}

func (jnt *FtpJoint) Close() (err error) {
	jnt.path = ""
	if jnt.ReadCloser != nil {
		err = jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = 0
	jnt.end = 0
	PutJoint(jnt)
	return
}

func (jnt *FtpJoint) Info(fpath string) (fi fs.FileInfo, err error) {
	var ent *ftp.Entry
	if ent, err = jnt.conn.GetEntry(path.Join(jnt.pwd, fpath)); err != nil {
		return
	}
	fi = FtpFileInfo{
		ent,
	}
	return
}

func (jnt *FtpJoint) ReadDir(fpath string) (ret []fs.FileInfo, err error) {
	fpath = FtpEscapeBrackets(path.Join(jnt.pwd, fpath))
	var entries []*ftp.Entry
	if entries, err = jnt.conn.List(fpath); err != nil {
		return
	}
	ret = make([]fs.FileInfo, 0, len(entries))
	for _, ent := range entries {
		if ent.Name != "." && ent.Name != ".." {
			ret = append(ret, FtpFileInfo{ent})
		}
	}
	return
}

func (jnt *FtpJoint) Stat() (fi fs.FileInfo, err error) {
	var ent *ftp.Entry
	if ent, err = jnt.conn.GetEntry(path.Join(jnt.pwd, jnt.path)); err != nil {
		return
	}
	fi = FtpFileInfo{
		ent,
	}
	return
}

func (jnt *FtpJoint) Size() int64 {
	if jnt.end == 0 {
		jnt.end, _ = jnt.conn.FileSize(path.Join(jnt.pwd, jnt.path))
	}
	return jnt.end
}

func (jnt *FtpJoint) Read(b []byte) (n int, err error) {
	if jnt.ReadCloser == nil {
		if jnt.ReadCloser, err = jnt.conn.RetrFrom(path.Join(jnt.pwd, jnt.path), uint64(jnt.pos)); err != nil {
			return
		}
	}
	n, err = jnt.ReadCloser.Read(b)
	jnt.pos += int64(n)
	return
}

func (jnt *FtpJoint) Write(p []byte) (n int, err error) {
	var buf = bytes.NewReader(p)
	err = jnt.conn.StorFrom(path.Join(jnt.pwd, jnt.path), buf, uint64(jnt.pos))
	var n64, _ = buf.Seek(0, io.SeekCurrent)
	jnt.pos += n64
	n = int(n64)
	return
}

func (jnt *FtpJoint) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = jnt.pos + offset
	case io.SeekEnd:
		if jnt.end == 0 {
			if jnt.end, err = jnt.conn.FileSize(path.Join(jnt.pwd, jnt.path)); err != nil {
				return
			}
		}
		abs = jnt.end + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
		return
	}
	if abs != jnt.pos && jnt.ReadCloser != nil {
		jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = abs
	return
}

func (jnt *FtpJoint) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		err = ErrFtpNegPos
		return
	}
	if off != jnt.pos && jnt.ReadCloser != nil {
		jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = off
	return jnt.Read(b)
}

// FtpFileInfo encapsulates ftp.Entry structure and provides fs.FileInfo implementation.
type FtpFileInfo struct {
	*ftp.Entry
}

// fs.FileInfo implementation.
func (fi FtpFileInfo) Name() string {
	return path.Base(fi.Entry.Name)
}

// fs.FileInfo implementation.
func (fi FtpFileInfo) Size() int64 {
	return int64(fi.Entry.Size)
}

// fs.FileInfo implementation.
func (fi FtpFileInfo) Mode() fs.FileMode {
	switch fi.Type {
	case ftp.EntryTypeFile:
		return 0444
	case ftp.EntryTypeFolder:
		return fs.ModeDir
	case ftp.EntryTypeLink:
		return fs.ModeSymlink
	}
	return 0
}

// fs.FileInfo implementation.
func (fi FtpFileInfo) ModTime() time.Time {
	return fi.Entry.Time
}

// fs.FileInfo implementation.
func (fi FtpFileInfo) IsDir() bool {
	return fi.Entry.Type == ftp.EntryTypeFolder
}

// fs.FileInfo implementation. Returns structure pointer itself.
func (fi FtpFileInfo) Sys() interface{} {
	return fi
}

// The End.
