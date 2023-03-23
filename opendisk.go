package hms

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/jlaffaye/ftp"
	"golang.org/x/text/encoding/charmap"
)

var (
	ErrFtpWhence = errors.New("invalid whence at FTP seeker")
	ErrFtpNegPos = errors.New("negative position at FTP seeker")
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

// FtpFileInfo encapsulates ftp.Entry structure and provides fs.FileInfo implementation.
type FtpFileInfo struct {
	*ftp.Entry
}

// fs.FileInfo implementation.
func (fi *FtpFileInfo) Name() string {
	return fi.Entry.Name
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

type FtpIoSeeker struct {
	path string
	conn *ftp.ServerConn
	resp *ftp.Response
	pos  int64
	end  int64
}

func (r *FtpIoSeeker) Close() (err error) {
	if r.resp != nil {
		err = r.resp.Close()
		r.resp = nil
	}
	return
}

func (r *FtpIoSeeker) Size() int64 {
	if r.end == 0 {
		r.end, _ = r.conn.FileSize(r.path)
	}
	return r.end
}

func (r *FtpIoSeeker) Read(b []byte) (n int, err error) {
	if r.pos >= r.Size() {
		err = io.EOF
	}
	if r.resp == nil {
		if r.resp, err = r.conn.RetrFrom(r.path, uint64(r.pos)); err != nil {
			return
		}
	}
	var written int64
	written, err = io.Copy(bytes.NewBuffer(b), r.resp)
	r.pos += written
	n = int(written)
	return
}

func (r *FtpIoSeeker) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.pos + offset
	case io.SeekEnd:
		abs = r.Size() + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
	}
	if abs != r.pos && r.resp != nil {
		err = r.resp.Close()
		r.resp = nil
		if err != nil {
			return
		}
	}
	r.pos = abs
	return
}

// OpenFile opens file from file system, or looking for iso-disk in the given path,
// opens it, and opens nested into iso-disk file.
func OpenFile(syspath string) (r io.ReadSeekCloser, err error) {
	if strings.HasPrefix(syspath, "ftp://") {
		var u *url.URL
		if u, err = url.Parse(syspath); err != nil {
			return
		}
		var conn *ftp.ServerConn
		if conn, err = FtpCacheGet((&url.URL{
			Scheme: u.Scheme,
			User:   u.User,
			Host:   u.Host,
		}).String()); err != nil {
			return
		}
		r = &FtpIoSeeker{
			path: u.Path,
			conn: conn,
		}
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
		if conn, err = FtpCacheGet((&url.URL{
			Scheme: u.Scheme,
			User:   u.User,
			Host:   u.Host,
		}).String()); err != nil {
			return
		}
		var ent *ftp.Entry
		if ent, err = conn.GetEntry(u.Path); err != nil {
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

// ReadDir returns directory files fs.FileInfo list. It scan file system path,
// or looking for iso-disk in the given path, opens it, and scan files nested
// into iso-disk local directory.
func ReadDir(dir string) (ret []fs.FileInfo, err error) {
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

// The End.
