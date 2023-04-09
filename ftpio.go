package hms

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
)

var (
	ErrFtpWhence = errors.New("invalid whence at FTP seeker")
	ErrFtpNegPos = errors.New("negative position at FTP seeker")
)

var (
	pwdmap map[string]string
	pwdmux sync.RWMutex
)

func PwdCache(ftpaddr string, conn *ftp.ServerConn) (pwd string) {
	var ok bool
	pwdmux.RLock()
	pwd, ok = pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if ok {
		return
	}
	var err error
	if pwd, err = conn.CurrentDir(); err == nil {
		pwdmux.Lock()
		pwdmap[ftpaddr] = pwd
		pwdmux.Unlock()
		return
	}
	return
}

// FtpFileInfo encapsulates ftp.Entry structure and provides fs.FileInfo implementation.
type FtpFileInfo struct {
	*ftp.Entry
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) Name() string {
	return path.Base(fi.Entry.Name)
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) Size() int64 {
	return int64(fi.Entry.Size)
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) Mode() fs.FileMode {
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
func (fi *FtpFileInfo) ModTime() time.Time {
	return fi.Entry.Time
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) IsDir() bool {
	return fi.Entry.Type == ftp.EntryTypeFolder
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) Sys() interface{} {
	return fi
}

// FtpFile implements for FTP-file io.Reader, io.Writer, io.Seeker, io.Closer.
type FtpFile struct {
	path string
	conn *ftp.ServerConn
	resp *ftp.Response
	pos  int64
	end  int64
}

// Opens new connection for any some one file with given full FTP URL.
// FTP-connection can serve only one file by the time, so it can not
// be used for parallel reading group of files.
func (ff *FtpFile) Open(ftppath string) (err error) {
	var u *url.URL
	if u, err = url.Parse(ftppath); err != nil {
		return
	}
	if ff.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
		return
	}
	ff.path = path.Join(PwdCache((&url.URL{
		Scheme: u.Scheme,
		User:   u.User,
		Host:   u.Host,
	}).String(), ff.conn), u.Path)
	var pass, _ = u.User.Password()
	if err = ff.conn.Login(u.User.Username(), pass); err != nil {
		return
	}
	ff.resp = nil
	ff.pos, ff.end = 0, 0
	return
}

func (ff *FtpFile) Close() (err error) {
	if ff.resp != nil {
		err = ff.resp.Close()
		ff.resp = nil
	}
	ff.conn.Quit()
	return
}

func (ff *FtpFile) Size() int64 {
	if ff.end == 0 {
		ff.end, _ = ff.conn.FileSize(ff.path)
	}
	return ff.end
}

func (ff *FtpFile) Read(b []byte) (n int, err error) {
	if ff.resp == nil {
		if ff.resp, err = ff.conn.RetrFrom(ff.path, uint64(ff.pos)); err != nil {
			return
		}
	}
	n, err = ff.resp.Read(b)
	ff.pos += int64(n)
	return
}

func (ff *FtpFile) Write(p []byte) (n int, err error) {
	var buf = bytes.NewReader(p)
	err = ff.conn.StorFrom(ff.path, buf, uint64(ff.pos))
	var n64, _ = buf.Seek(0, io.SeekCurrent)
	ff.pos += n64
	n = int(n64)
	return
}

func (ff *FtpFile) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = ff.pos + offset
	case io.SeekEnd:
		if ff.end == 0 {
			if ff.end, err = ff.conn.FileSize(ff.path); err != nil {
				return
			}
		}
		abs = ff.end + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
	}
	if abs != ff.pos && ff.resp != nil {
		ff.resp.Close()
		ff.resp = nil
	}
	ff.pos = abs
	return
}

// The End.
