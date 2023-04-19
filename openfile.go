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
// opens it, and opens nested into iso-disk file.
func OpenFile(syspath string) (r File, err error) {
	if strings.HasPrefix(syspath, "ftp://") {
		var f FtpFile
		if err = f.Open(syspath); err != nil {
			return
		}
		r = &f
		return
	} else {
		if r, err = os.Open(syspath); err == nil { // primary filesystem file
			return
		}
		var file io.Closer = io.NopCloser(nil) // empty closer stub

		// looking for nested file
		var isopath = syspath
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == syspath {
			fpath = "/" // get root of disk
		} else {
			fpath = syspath[len(isopath):]
		}

		var f IsoFile
		if err = f.Open(isopath, fpath); err != nil {
			return
		}
		r = &f
		return
	}
}

// StatFile returns fs.FileInfo of file in file system, or file nested in disk image.
func StatFile(syspath string) (fi fs.FileInfo, err error) {
	if strings.HasPrefix(syspath, "ftp://") {
		var ftpaddr, ftppath = SplitUrl(syspath)
		var d *FtpJoint
		if d, err = GetFtpJoint(ftpaddr); err != nil {
			return
		}
		defer PutFtpJoint(ftpaddr, d)

		return d.Stat(ftppath)
	} else {
		// check up file is at primary filesystem
		var file *os.File
		if file, err = os.Open(syspath); err == nil {
			defer file.Close()
			return file.Stat()
		}

		// looking for nested file
		var isopath = syspath
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == syspath {
			fpath = "/" // get root of disk
		} else {
			fpath = syspath[len(isopath):]
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
// into iso-disk local directory.
func ReadDir(dir string) (ret []fs.FileInfo, err error) {
	if strings.HasPrefix(dir, "ftp://") {
		var ftpaddr, ftppath = SplitUrl(dir)
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
	} else {
		var file *os.File
		if file, err = os.Open(dir); err == nil { // primary filesystem file
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
		var isopath = dir
		for errors.Is(err, fs.ErrNotExist) && isopath != "." && isopath != "/" {
			isopath = path.Dir(isopath)
			file, err = os.Open(isopath)
		}
		if err != nil {
			return
		}
		file.Close()

		var fpath string
		if isopath == dir {
			fpath = "/" // get root of disk
		} else {
			fpath = dir[len(isopath):]
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
