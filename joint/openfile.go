package joint

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/jlaffaye/ftp"
	iso "github.com/kdomanski/iso9660"
)

// RFile combines fs.File interface and io.Seeker interface.
type RFile interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	fs.File
}

// OpenFile opens file from file system, or looking for iso-disk in the given path,
// opens it, and opens nested into iso-disk file. Or opens file at cloud.
func OpenFile(anypath string) (r RFile, err error) {
	if strings.HasPrefix(anypath, "ftp://") {
		var f FtpFile
		if err = f.Open(anypath); err != nil {
			return
		}
		r = &f
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var f SftpFile
		if err = f.Open(anypath); err != nil {
			return
		}
		r = &f
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var f DavFile
		if err = f.Open(anypath); err != nil {
			return
		}
		r = &f
		return
	} else {
		if r, err = os.Open(anypath); err == nil { // primary filesystem file
			return
		}
		var file io.Closer = io.NopCloser(nil) // empty closer stub

		// looking for nested file
		var isopath = anypath
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == anypath {
			fpath = "" // get root of disk
		} else {
			fpath = anypath[len(isopath)+1:] // without slash prefix
		}

		var f IsoFile
		if err = f.Open(isopath, fpath); err != nil {
			return
		}
		r = &f
		return
	}
}

// StatFile returns fs.FileInfo of file in file system,
// or file nested in disk image, or cloud file.
func StatFile(anypath string) (fi fs.FileInfo, err error) {
	if strings.HasPrefix(anypath, "ftp://") {
		var ftpaddr, ftppath = SplitUrl(anypath)
		var d *FtpJoint
		if d, err = GetFtpJoint(ftpaddr); err != nil {
			return
		}

		fi, err = d.Stat(ftppath)
		if err != nil { // on case connection was dropped
			d.Close()
		} else {
			PutFtpJoint(ftpaddr, d)
		}
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var sftpaddr, sftppath = SplitUrl(anypath)
		var d *SftpJoint
		if d, err = GetSftpJoint(sftpaddr); err != nil {
			return
		}

		fi, err = d.Stat(sftppath)
		if err != nil { // on case connection was dropped
			d.Close()
		} else {
			PutSftpJoint(sftpaddr, d)
		}
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var addr, fpath, ok = GetDavPath(anypath)
		if !ok {
			err = ErrNotFound
			return
		}

		var d *DavJoint
		if d, err = GetDavJoint(addr); err != nil {
			return
		}

		fi, err = d.Stat(fpath)
		if err != nil { // on case connection was dropped
			d.Close()
		} else {
			PutDavJoint(addr, d)
		}
		return
	} else {
		// check up file is at primary filesystem
		var file *os.File
		if file, err = os.Open(anypath); err == nil {
			defer file.Close()
			return file.Stat()
		}

		// looking for nested file
		var isopath = anypath
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == anypath {
			fpath = "" // get root of disk
		} else {
			fpath = anypath[len(isopath)+1:] // without slash prefix
		}

		var d *IsoJoint
		if d, err = GetIsoJoint(isopath); err != nil {
			return
		}

		fi, err = d.Stat(fpath)
		if err != nil { // on case connection was dropped
			d.Close()
		} else {
			PutIsoJoint(isopath, d)
		}
		return
	}
}

// FtpEscapeBrackets escapes square brackets at FTP-path.
// FTP-server does not recognize path with square brackets
// as is to get a list of files, so such path should be escaped.
func FtpEscapeBrackets(s string) string {
	var n = 0
	for _, c := range s {
		if c == '[' || c == ']' {
			n++
		}
	}
	if n == 0 {
		return s
	}
	var esc = make([]rune, 0, len(s)+n*2)
	for _, c := range s {
		if c == '[' {
			esc = append(esc, '[', '[', ']')
		} else if c == ']' {
			esc = append(esc, '[', ']', ']')
		} else {
			esc = append(esc, c)
		}
	}
	return string(esc)
}

// ReadDir returns directory files fs.FileInfo list. It scan file system path,
// or looking for iso-disk in the given path, opens it, and scan files nested
// into iso-disk local directory. Or reads directory at cloud path.
func ReadDir(anypath string) (ret []fs.FileInfo, err error) {
	if strings.HasPrefix(anypath, "ftp://") {
		var ftpaddr, ftppath = SplitUrl(anypath)
		var d *FtpJoint
		if d, err = GetFtpJoint(ftpaddr); err != nil {
			return
		}
		defer PutFtpJoint(ftpaddr, d)

		var fpath = FtpEscapeBrackets(path.Join(d.pwd, ftppath))
		var entries []*ftp.Entry
		if entries, err = d.conn.List(fpath); err != nil {
			return
		}
		ret = make([]fs.FileInfo, 0, len(entries))
		for _, ent := range entries {
			if ent.Name != "." && ent.Name != ".." {
				ret = append(ret, FtpFileInfo{ent})
			}
		}
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var sftpaddr, sftppath = SplitUrl(anypath)
		var d *SftpJoint
		if d, err = GetSftpJoint(sftpaddr); err != nil {
			return
		}
		defer PutSftpJoint(sftpaddr, d)

		var fpath = path.Join(d.pwd, sftppath)
		if ret, err = d.client.ReadDir(fpath); err != nil {
			return
		}
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var addr, fpath, ok = GetDavPath(anypath)
		if !ok {
			err = ErrNotFound
			return
		}

		var d *DavJoint
		if d, err = GetDavJoint(addr); err != nil {
			return
		}
		defer PutDavJoint(addr, d)

		if ret, err = d.client.ReadDir(fpath); err != nil {
			return
		}
		return
	} else {
		var file *os.File
		if file, err = os.Open(anypath); err == nil { // primary filesystem file
			defer file.Close()
			var fi fs.FileInfo
			if fi, err = file.Stat(); err != nil {
				return
			}
			if fi.IsDir() { // get the list only for directory
				return file.Readdir(-1)
			}
		}

		// looking for nested file
		var isopath = anypath
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == anypath {
			fpath = "" // get root of disk
		} else {
			fpath = anypath[len(isopath)+1:] // without slash prefix
		}

		var d *IsoJoint
		if d, err = GetIsoJoint(isopath); err != nil {
			return
		}
		defer PutIsoJoint(isopath, d)

		var f IsoFile
		if err = f.Open(isopath, fpath); err != nil {
			return
		}
		var files []*iso.File
		if files, err = f.GetChildren(); err != nil {
			return
		}
		ret = make([]fs.FileInfo, len(files))
		for i, file := range files {
			ret[i] = &IsoFile{
				File: file,
			}
		}
		return
	}
}

type FS string

// joinfast performs fast join of two path chunks.
func joinfast(dir, base string) string {
	if dir == "" || dir == "." {
		return base
	}
	if base == "" || base == "." {
		return dir
	}
	if dir[len(dir)-1] == '/' {
		if base[0] == '/' {
			return dir + base[1:]
		} else {
			return dir + base
		}
	}
	if base[0] == '/' {
		return dir + base
	}
	return dir + "/" + base
}

func (fsys FS) Open(fpath string) (r fs.File, err error) {
	return OpenFile(joinfast(string(fsys), fpath))
}

func (fsys FS) Stat() (fi fs.FileInfo, err error) {
	return StatFile(string(fsys))
}

func (fsys FS) ReadDir(fpath string) (ret []fs.DirEntry, err error) {
	var fis []fs.FileInfo
	if fis, err = ReadDir(joinfast(string(fsys), fpath)); err != nil {
		return
	}
	ret = make([]fs.DirEntry, len(fis))
	for i, fi := range fis {
		ret[i] = fs.FileInfoToDirEntry(fi)
	}
	return
}

// The End.
