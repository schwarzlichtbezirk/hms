package hms

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/jlaffaye/ftp"
	"golang.org/x/text/encoding/charmap"
)

// OpenFile opens file from file system, or looking for iso-disk in the given path,
// opens it, and opens nested into iso-disk file.
func OpenFile(syspath string) (r io.ReadSeekCloser, err error) {
	if strings.HasPrefix(syspath, "ftp://") {
		var ff FtpFile
		if err = ff.Open(syspath); err != nil {
			return
		}
		r = &ff
		return
	} else {
		if r, err = os.Open(syspath); err == nil { // primary filesystem file
			return
		}
		var file io.Closer = io.NopCloser(nil) // empty closer stub

		// looking for nested file
		var basepath = syspath
		var operr = err
		for errors.Is(operr, fs.ErrNotExist) && basepath != "." && basepath != "/" {
			basepath = path.Dir(basepath)
			file, operr = os.Open(basepath)
		}
		if operr != nil {
			err = operr
			return
		}
		file.Close()

		var disk *DiskFS
		if disk, err = DiskCacheGet(basepath); err != nil {
			return
		}

		var diskpath = syspath[len(basepath):]
		return disk.OpenFile(diskpath)
	}
}

// StatFile returns fs.FileInfo of file in file system, or file nested in disk image.
func StatFile(syspath string) (fi fs.FileInfo, err error) {
	if strings.HasPrefix(syspath, "ftp://") {
		var ftpaddr, ftppath = SplitUrl(syspath)
		var conn *ftp.ServerConn
		if conn, err = GetFtpConn(ftpaddr); err != nil {
			return
		}
		defer PutFtpConn(ftpaddr, conn)

		var fpath = FtpPwdPath(ftpaddr, ftppath, conn)
		var ent *ftp.Entry
		if ent, err = conn.GetEntry(fpath); err != nil {
			return
		}
		fi = &FtpFileInfo{
			ent,
		}
		return
	} else {
		// check up file is at primary filesystem
		var file *os.File
		if file, err = os.Open(syspath); err == nil {
			defer file.Close()
			return file.Stat()
		}

		// looking for nested file
		var basepath = syspath
		for errors.Is(err, fs.ErrNotExist) && basepath != "." && basepath != "/" {
			basepath = path.Dir(basepath)
			file, err = os.Open(basepath)
		}
		if err != nil {
			return
		}
		file.Close()

		var disk *DiskFS
		if disk, err = DiskCacheGet(basepath); err != nil {
			return
		}

		var diskpath = syspath[len(basepath):]
		var _, isiso = disk.fs.(*iso9660.FileSystem)
		if isiso {
			var enc = charmap.Windows1251.NewEncoder()
			diskpath, _ = enc.String(diskpath)
		}

		var list []fs.FileInfo
		if list, err = disk.fs.ReadDir(path.Dir(diskpath)); err != nil {
			return
		}

		var finame = path.Base(diskpath)
		for _, fi = range list {
			if fi.Name() == finame {
				if isiso {
					return &IsoFileInfo{fi}, nil
				} else {
					return fi, nil
				}
			}
		}
		return nil, ErrNotFound
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
		var conn *ftp.ServerConn
		if conn, err = GetFtpConn(ftpaddr); err != nil {
			return
		}
		defer PutFtpConn(ftpaddr, conn)

		var fpath = FtpEscapeBrackets(FtpPwdPath(ftpaddr, ftppath, conn))
		var entries []*ftp.Entry
		if entries, err = conn.List(fpath); err != nil {
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
		var basepath = dir
		var operr = err
		for errors.Is(operr, fs.ErrNotExist) && basepath != "." && basepath != "/" {
			basepath = path.Dir(basepath)
			file, operr = os.Open(basepath)
		}
		if operr != nil {
			err = operr
			return
		}
		file.Close()

		var disk *DiskFS
		if disk, err = DiskCacheGet(basepath); err != nil {
			return
		}

		var diskpath string
		if basepath == dir {
			diskpath = "/" // list root of disk
		} else {
			diskpath = dir[len(basepath):]
		}
		var _, isiso = disk.fs.(*iso9660.FileSystem)
		if isiso {
			var enc = charmap.Windows1251.NewEncoder()
			diskpath, _ = enc.String(diskpath)
		}
		if ret, err = disk.fs.ReadDir(diskpath); err != nil {
			return
		}
		if isiso {
			for i, fi := range ret {
				ret[i] = &IsoFileInfo{fi}
			}
		}
		return
	}
}

// The End.
