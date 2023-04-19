package hms

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/diskfs/go-diskfs"
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

// OpenFile opens nested into iso-disk file with given local path from iso-disk root.
func (d *DiskFS) OpenFile(fpath string) (r io.ReadSeekCloser, err error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	var _, isiso = d.fs.(*iso9660.FileSystem)
	if isiso {
		var enc = charmap.Windows1251.NewEncoder()
		fpath, _ = enc.String(fpath)
	}

	if r, err = d.fs.OpenFile(fpath, os.O_RDONLY); err != nil {
		return
	}
	return
}

type IsoFile struct {
	isofile string
	fpath   string
	d       *IsoJoint
	io.ReadSeekCloser
}

func (f *IsoFile) Open(isofile, fpath string) (err error) {
	f.isofile, f.fpath = isofile, fpath
	if f.d, err = GetIsoJoint(isofile); err != nil {
		return
	}
	var enc = charmap.Windows1251.NewEncoder()
	fpath, _ = enc.String(fpath)
	if f.ReadSeekCloser, err = f.d.fs.OpenFile(fpath, os.O_RDONLY); err != nil {
		return
	}
	return
}

func (f *IsoFile) Close() (err error) {
	err = f.ReadSeekCloser.Close()
	PutIsoJoint(f.isofile, f.d)
	return
}

func (f *IsoFile) Stat() (fi fs.FileInfo, err error) {
	var enc = charmap.Windows1251.NewEncoder()
	var fpath, _ = enc.String(f.fpath)

	var list []fs.FileInfo
	if list, err = f.d.fs.ReadDir(path.Dir(fpath)); err != nil {
		return
	}

	var fname = path.Base(fpath)
	for _, fi = range list {
		if fi.Name() == fname {
			return &IsoFileInfo{fi}, nil
		}
	}
	return nil, ErrNotFound
}

// IsoFileInfo is wrapper to convert file names in code page 1251 to UTF.
type IsoFileInfo struct {
	fs.FileInfo
}

func (fi *IsoFileInfo) Name() string {
	var dec = charmap.Windows1251.NewDecoder()
	var name, _ = dec.String(fi.FileInfo.Name())
	return name
}

// The End.
