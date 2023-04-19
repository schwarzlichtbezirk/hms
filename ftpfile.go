package hms

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
	"golang.org/x/crypto/ssh"
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

// FtpPwdPath return path from given URL concatenated with FTP
// current directory. It's used cache to avoid extra calls to
// FTP-server to get current directory for every call.
func FtpPwdPath(ftpaddr, ftppath string, conn *ftp.ServerConn) (fpath string) {
	pwdmux.RLock()
	var pwd, ok = pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if !ok {
		var err error
		if pwd, err = conn.CurrentDir(); err == nil {
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	fpath = path.Join(pwd, ftppath)
	if strings.HasPrefix(fpath, "/") {
		fpath = fpath[1:]
	}
	return
}

// FtpPwdPath return path from given URL concatenated with SFTP
// current directory. It's used cache to avoid extra calls to
// FTP-server to get current directory for every call.
func SftpPwdPath(ftpaddr, ftppath string, client *sftp.Client) (fpath string) {
	pwdmux.RLock()
	var pwd, ok = pwdmap[ftpaddr]
	pwdmux.RUnlock()
	if !ok {
		var err error
		if pwd, err = client.Getwd(); err == nil {
			pwdmux.Lock()
			pwdmap[ftpaddr] = pwd
			pwdmux.Unlock()
		}
	}
	fpath = path.Join(pwd, ftppath)
	return
}

func SftpOpenFile(ftpurl string) (r io.ReadSeekCloser, err error) {
	var ftpaddr, ftppath = SplitUrl(ftpurl)
	var conn *ssh.Client
	var config = &ssh.ClientConfig{
		User: "music",
		Auth: []ssh.AuthMethod{
			ssh.Password("x"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if conn, err = ssh.Dial("tcp", "192.168.1.1:22", config); err != nil {
		return
	}
	defer conn.Close()

	var client *sftp.Client
	if client, err = sftp.NewClient(conn); err != nil {
		return
	}
	defer client.Close()

	var fpath = SftpPwdPath(ftpaddr, ftppath, client)
	var f *sftp.File
	if f, err = client.Open(fpath); err != nil {
		return
	}
	r = f
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
	addr string
	path string
	conn *ftp.ServerConn
	resp *ftp.Response
	pos  int64
	end  int64
}

// Opens new connection for any some one file with given full FTP URL.
// FTP-connection can serve only one file by the time, so it can not
// be used for parallel reading group of files.
func (ff *FtpFile) Open(ftpurl string) (err error) {
	var ftpaddr, ftppath = SplitUrl(ftpurl)
	if ff.conn, err = GetFtpConn(ftpaddr); err != nil {
		return
	}
	ff.addr = ftpaddr
	ff.path = FtpPwdPath(ftpaddr, ftppath, ff.conn)
	ff.resp = nil
	ff.pos, ff.end = 0, 0
	return
}

func (ff *FtpFile) Close() (err error) {
	if ff.resp != nil {
		err = ff.resp.Close()
		ff.resp = nil
	}
	PutFtpConn(ff.addr, ff.conn)
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
