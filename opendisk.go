package hms

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/jlaffaye/ftp"
	"golang.org/x/text/encoding/charmap"
)

// DiskFS is ISO or FAT32 disk structure representation for quick access to nested files.
// This structures can be cached and closed on cache expiration.
type DiskFS struct {
	file *os.File
	fs   filesystem.FileSystem
	mux  sync.Mutex
}

// NewDiskFS creates new DiskFS with opened disk image by given path.
func NewDiskFS(fpath string) (d *DiskFS, err error) {
	d = &DiskFS{}
	var disk *disk.Disk
	if disk, err = diskfs.Open(fpath, diskfs.WithOpenMode(diskfs.ReadOnly)); err != nil {
		return
	}
	d.file = disk.File
	if d.fs, err = disk.GetFilesystem(0); err != nil { // assuming it is the whole disk, so partition = 0
		disk.File.Close()
		return
	}
	return
}

// Close performs to close iso-disk file.
func (d *DiskFS) Close() error {
	d.mux.Lock()
	defer d.mux.Unlock()

	return d.file.Close()
}

type nopCloserReadSeek struct {
	io.ReadSeeker
}

func (f *nopCloserReadSeek) Close() error {
	return nil
}

// OpenFile opens nested into iso-disk file with given local path from iso-disk root.
func (d *DiskFS) OpenFile(fpath string) (r io.ReadSeekCloser, err error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	var _, isiso = d.fs.(*iso9660.FileSystem)
	if isiso {
		var enc = charmap.Windows1251.NewEncoder()
		fpath, _ = enc.String(fpath)
	}

	var file filesystem.File
	if file, err = d.fs.OpenFile(fpath, os.O_RDONLY); err != nil {
		return
	}
	r = &nopCloserReadSeek{file}
	return
}

// FileInfoISO is wrapper to convert file names in code page 1251 to UTF.
type FileInfoISO struct {
	fs.FileInfo
}

func (fi *FileInfoISO) Name() string {
	var dec = charmap.Windows1251.NewDecoder()
	var name, _ = dec.String(fi.FileInfo.Name())
	return name
}

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
		var file io.Closer = &nopCloserReadSeek{}

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
		var u *url.URL
		if u, err = url.Parse(syspath); err != nil {
			return
		}
		var conn *ftp.ServerConn
		if conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
			return
		}
		defer conn.Quit()
		var pass, _ = u.User.Password()
		if err = conn.Login(u.User.Username(), pass); err != nil {
			return
		}
		var fpath = FtpPwdPath(syspath, conn)
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
					return &FileInfoISO{fi}, nil
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
		var u *url.URL
		if u, err = url.Parse(dir); err != nil {
			return
		}
		var conn *ftp.ServerConn
		if conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
			return
		}
		defer conn.Quit()
		var pass, _ = u.User.Password()
		if err = conn.Login(u.User.Username(), pass); err != nil {
			return
		}
		var fpath = FtpEscapeBrackets(FtpPwdPath(dir, conn))
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
				ret[i] = &FileInfoISO{fi}
			}
		}
		return
	}
}

// The End.
