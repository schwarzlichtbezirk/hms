package joint

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
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
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &FtpJoint{}); err != nil {
			return
		}
		if _, err = jnt.Open(fpath); err != nil {
			return
		}
		r = jnt
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &SftpJoint{}); err != nil {
			return
		}
		if _, err = jnt.Open(fpath); err != nil {
			return
		}
		r = jnt
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var addr, fpath, ok = GetDavPath(anypath)
		if !ok {
			err = ErrNotFound
			return
		}
		var jnt Joint
		if jnt, err = GetJoint(addr, &DavJoint{}); err != nil {
			return
		}

		if _, err = jnt.Open(fpath); err != nil {
			return
		}
		r = jnt
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

		var jnt Joint
		if jnt, err = GetJoint(isopath, &IsoJoint{}); err != nil {
			return
		}
		if _, err = jnt.Open(fpath); err != nil {
			return
		}
		r = jnt
		return
	}
}

/*func OpenNested(anypath string) (r RFile, err error) {
	var chunks = strings.Split(anypath, "/")
	var curdir, curpath string
	for i, chunk := range chunks {
		curdir = curpath
		if i > 0 {
			curpath += "/"
		}
		curpath += chunk
		if strings.ToLower(path.Ext(chunk)) == ".iso" {
			var fi fs.FileInfo
			if fi, err = os.Stat(curpath); err != nil {
				return
			}
			if fi.IsDir() {
				continue
			}
		}
	}
}*/

// StatFile returns fs.FileInfo of file in file system,
// or file nested in disk image, or cloud file.
func StatFile(anypath string) (fi fs.FileInfo, err error) {
	if strings.HasPrefix(anypath, "ftp://") {
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &FtpJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		fi, err = jnt.Info(fpath)
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &SftpJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		fi, err = jnt.Info(fpath)
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var addr, fpath, ok = GetDavPath(anypath)
		if !ok {
			err = ErrNotFound
			return
		}
		var jnt Joint
		if jnt, err = GetJoint(addr, &DavJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		fi, err = jnt.Info(fpath)
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

		var jnt Joint
		if jnt, err = GetJoint(isopath, &IsoJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		fi, err = jnt.Info(fpath)
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
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &FtpJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		ret, err = jnt.ReadDir(fpath)
		return
	} else if strings.HasPrefix(anypath, "sftp://") {
		var addr, fpath = SplitUrl(anypath)
		var jnt Joint
		if jnt, err = GetJoint(addr, &SftpJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		ret, err = jnt.ReadDir(fpath)
		return
	} else if strings.HasPrefix(anypath, "http://") || strings.HasPrefix(anypath, "https://") {
		var addr, fpath, ok = GetDavPath(anypath)
		if !ok {
			err = ErrNotFound
			return
		}
		var jnt Joint
		if jnt, err = GetJoint(addr, &DavJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		ret, err = jnt.ReadDir(fpath)
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

		var jnt Joint
		if jnt, err = GetJoint(isopath, &IsoJoint{}); err != nil {
			return
		}
		defer func() {
			if err != nil {
				jnt.Cleanup()
			} else {
				PutJoint(jnt)
			}
		}()

		ret, err = jnt.ReadDir(fpath)
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
