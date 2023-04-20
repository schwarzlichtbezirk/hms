package hms

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/jlaffaye/ftp"
	"golang.org/x/text/encoding/charmap"
)

// OpenFile opens file from file system, or looking for iso-disk in the given path,
// opens it, and opens nested into iso-disk file. Or opens file at cloud.
func OpenFile(anypath string) (r File, err error) {
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
			fpath = "/" // get root of disk
		} else {
			fpath = anypath[len(isopath):]
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
		defer PutFtpJoint(ftpaddr, d)

		return d.Stat(ftppath)
	} else if strings.HasPrefix(anypath, "sftp://") {
		var sftpaddr, sftppath = SplitUrl(anypath)
		var d *SftpJoint
		if d, err = GetSftpJoint(sftpaddr); err != nil {
			return
		}
		defer PutSftpJoint(sftpaddr, d)

		return d.Stat(sftppath)
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
			fpath = "/" // get root of disk
		} else {
			fpath = anypath[len(isopath):]
		}

		var d *IsoJoint
		if d, err = GetIsoJoint(isopath); err != nil {
			return
		}
		defer PutIsoJoint(isopath, d)

		return d.Stat(fpath)
	}
}

// FtpEscapeBrackets escapes square brackets at FTP-path.
// FTP-server does not recognize path with square brackets
// as is to get a list of files, so such path should be escaped.
func FtpEscapeBrackets(s string) string {
	var b = s2b(s)
	var n = 0
	for _, c := range b {
		if c == '[' || c == ']' {
			n++
		}
	}
	if n == 0 {
		return s
	}
	var esc = make([]byte, 0, len(b)+n*2)
	for _, c := range b {
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
				ret = append(ret, &FtpFileInfo{ent})
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

		var fpath = FtpEscapeBrackets(path.Join(d.pwd, sftppath))
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
			fpath = "/" // get root of disk
		} else {
			fpath = anypath[len(isopath):]
		}

		var d *IsoJoint
		if d, err = GetIsoJoint(isopath); err != nil {
			return
		}
		defer PutIsoJoint(isopath, d)

		var enc = charmap.Windows1251.NewEncoder()
		fpath, _ = enc.String(fpath)
		if ret, err = d.fs.ReadDir(fpath); err != nil {
			return
		}
		for i, fi := range ret {
			ret[i] = &IsoFileInfo{fi}
		}
		return
	}
}

// The End.
