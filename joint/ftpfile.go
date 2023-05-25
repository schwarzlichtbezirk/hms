package joint

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
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
			if strings.HasPrefix(pwd, "/") {
				pwd = pwd[1:]
			}
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	return
}

// SftpPwd return SFTP current directory. It's used cache to avoid
// extra calls to SFTP-server to get current directory for every call.
func SftpPwd(ftpaddr string, client *sftp.Client) (pwd string) {
	pwdmux.RLock()
	pwd, ok := pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if !ok {
		var err error
		if pwd, err = client.Getwd(); err == nil {
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	return
}

// FtpFile implements for FTP-file io.Reader, io.Writer, io.Seeker, io.Closer.
type FtpFile struct {
	addr string // address of FTP-service, i.e. ftp://user:pass@example.com
	path string // path inside of FTP-service
	d    *FtpJoint
	io.ReadCloser
	pos int64
	end int64
}

// Opens new connection for any some one file with given full FTP URL.
// FTP-connection can serve only one file by the time, so it can not
// be used for parallel reading group of files.
func (f *FtpFile) Open(ftpurl string) (err error) {
	f.addr, f.path = SplitUrl(ftpurl)
	if f.d, err = GetFtpJoint(f.addr); err != nil {
		return
	}
	f.ReadCloser = nil
	f.pos, f.end = 0, 0
	return
}

func (f *FtpFile) Close() (err error) {
	if f.ReadCloser != nil {
		err = f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	PutFtpJoint(f.addr, f.d)
	return
}

func (f *FtpFile) Stat() (fi fs.FileInfo, err error) {
	var ent *ftp.Entry
	if ent, err = f.d.conn.GetEntry(path.Join(f.d.pwd, f.path)); err != nil {
		return
	}
	fi = FtpFileInfo{
		ent,
	}
	return
}

func (f *FtpFile) Size() int64 {
	if f.end == 0 {
		f.end, _ = f.d.conn.FileSize(path.Join(f.d.pwd, f.path))
	}
	return f.end
}

func (f *FtpFile) Read(b []byte) (n int, err error) {
	if f.ReadCloser == nil {
		if f.ReadCloser, err = f.d.conn.RetrFrom(path.Join(f.d.pwd, f.path), uint64(f.pos)); err != nil {
			return
		}
	}
	n, err = f.ReadCloser.Read(b)
	f.pos += int64(n)
	return
}

func (f *FtpFile) Write(p []byte) (n int, err error) {
	var buf = bytes.NewReader(p)
	err = f.d.conn.StorFrom(path.Join(f.d.pwd, f.path), buf, uint64(f.pos))
	var n64, _ = buf.Seek(0, io.SeekCurrent)
	f.pos += n64
	n = int(n64)
	return
}

func (f *FtpFile) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.pos + offset
	case io.SeekEnd:
		if f.end == 0 {
			if f.end, err = f.d.conn.FileSize(path.Join(f.d.pwd, f.path)); err != nil {
				return
			}
		}
		abs = f.end + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
	}
	if abs != f.pos && f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	f.pos = abs
	return
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

// SftpFile implements for SFTP-file io.Reader, io.Writer, io.Seeker, io.Closer.
type SftpFile struct {
	addr string // address of SFTP-service, i.e. sftp://user:pass@example.com
	path string // path inside of SFTP-service without PWD
	d    *SftpJoint
	*sftp.File
}

// Opens new connection for any some one file with given full SFTP URL.
func (f *SftpFile) Open(sftpurl string) (err error) {
	f.addr, f.path = SplitUrl(sftpurl)
	if f.d, err = GetSftpJoint(f.addr); err != nil {
		return
	}
	if f.File, err = f.d.client.Open(path.Join(f.d.pwd, f.path)); err != nil {
		return
	}
	return
}

func (f *SftpFile) Close() (err error) {
	err = f.File.Close()
	PutSftpJoint(f.addr, f.d)
	return
}

// The End.
