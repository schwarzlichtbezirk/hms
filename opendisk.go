package hms

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
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
	if disk, err = diskfs.OpenWithMode(fpath, diskfs.ReadOnly); err != nil {
		return
	}
	d.file = disk.File
	if d.fs, err = disk.GetFilesystem(0); err != nil { // assuming it is the whole disk, so partition = 0
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

	var enc = charmap.Windows1251.NewEncoder()
	fpath, _ = enc.String(fpath)

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

	var dv any
	if dv, operr = diskcache.Get(basepath); operr != nil {
		if !errors.Is(operr, ErrNotDisk) {
			err = operr
		}
		return
	}
	if err = diskcache.Set(basepath, dv); err != nil { // update expiration time
		return
	}

	var diskpath = syspath[len(basepath):]
	switch disk := dv.(type) {
	case *DiskFS:
		return disk.OpenFile(diskpath)
	}
	panic("not released disk type present")
}

// StatFile returns fs.FileInfo of file in file system, or file nested in disk image.
func StatFile(syspath string) (fi fs.FileInfo, err error) {
	var file io.ReadSeekCloser
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	switch file := file.(type) {
	case *os.File:
		return file.Stat()
	case filesystem.File:
		switch df := file.(type) {
		case *iso9660.File:
			return &FileInfoISO{df}, nil
		default:
			panic("not released disk type present")
		}
	case *nopCloserReadSeek:
		switch df := file.ReadSeeker.(type) {
		case *iso9660.File:
			return &FileInfoISO{df}, nil
		default:
			panic("not released disk type present")
		}
	default:
		panic("not released disk type present")
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

	var dv any
	if dv, operr = diskcache.Get(basepath); operr != nil {
		if !errors.Is(operr, ErrNotDisk) {
			err = operr
		}
		return
	}
	if err = diskcache.Set(basepath, dv); err != nil { // update expiration time
		return
	}

	var diskpath string
	if basepath == dir {
		diskpath = "/" // list root of disk
	} else {
		diskpath = dir[len(basepath):]
	}
	switch disk := dv.(type) {
	case *DiskFS:
		var enc = charmap.Windows1251.NewEncoder()
		diskpath, _ = enc.String(diskpath)
		if ret, err = disk.fs.ReadDir(diskpath); err != nil {
			return
		}
		for i, fi := range ret {
			ret[i] = &FileInfoISO{fi}
		}
		return
	}
	panic("not released disk type present")
}

// The End.
