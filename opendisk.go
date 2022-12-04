package hms

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"golang.org/x/text/encoding/charmap"
)

// DiskISO is iso-disk structure representation for quick access to nested files.
// This structures can be cached and closed on cache expiration.
type DiskISO struct {
	file *os.File
	fs   filesystem.FileSystem
	mux  sync.Mutex
}

// NewDiskISO creates new DiskISO with opened disk image by given path.
func NewDiskISO(fpath string) (d *DiskISO, err error) {
	d = &DiskISO{}
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
func (d *DiskISO) Close() error {
	d.mux.Lock()
	defer d.mux.Unlock()

	return d.file.Close()
}

type cfile struct {
	io.ReadSeeker
}

func (f *cfile) Close() error {
	return nil
}

// OpenFile opens nested into iso-disk file with given local path from iso-disk root.
func (d *DiskISO) OpenFile(fpath string) (r io.ReadSeekCloser, err error) {
	d.mux.Lock()
	defer d.mux.Unlock()

	var enc = charmap.Windows1251.NewEncoder()
	fpath, _ = enc.String(fpath)

	var file filesystem.File
	if file, err = d.fs.OpenFile(fpath, os.O_RDONLY); err != nil {
		return
	}
	r = &cfile{file}
	return
}

// OpenFile opens file from file system, or looking for iso-disk in the given path,
// opens it, and opens nested into iso-disk file.
func OpenFile(syspath string) (r io.ReadSeekCloser, err error) {
	var fpath = syspath
	// append slash to disk root to prevent open current dir on this disk
	if strings.HasSuffix(fpath, ":") {
		fpath += "/"
	}

	if r, err = os.Open(fpath); err == nil { // primary filesystem file
		return
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return
	}

	// looking for nested file
	var operr = err
	for operr != nil && fpath != "." && fpath != "/" {
		fpath = path.Dir(fpath)
		r, operr = os.Open(fpath)
	}
	if operr == nil {
		r.Close()
		r = nil
	}

	var dv interface{}
	if dv, operr = diskcache.Get(fpath); operr != nil {
		if !errors.Is(operr, ErrNotDisk) {
			err = operr
		}
		return
	}
	if err = diskcache.Set(fpath, dv); err != nil { // update expiration time
		return
	}

	var dpath = syspath[len(fpath):]
	switch disk := dv.(type) {
	case *DiskISO:
		return disk.OpenFile(dpath)
	}
	panic("not released disk type present")
}

// StatFile returns fs.FileInfo of file in file system, or file nested in disk image.
func StatFile(syspath string) (fi fs.FileInfo, err error) {
	var r io.ReadSeekCloser
	if r, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer r.Close()

	switch file := r.(type) {
	case *os.File:
		return file.Stat()
	case *cfile:
		switch df := file.ReadSeeker.(type) {
		case *iso9660.File:
			return df, nil
		default:
			panic("not released disk type present")
		}
	default:
		panic("not released disk type present")
	}
}

// OpenDir returns directory files fs.FileInfo list. It scan file system path,
// or looking for iso-disk in the given path, opens it, and scan files nested
// into iso-disk local directory.
func OpenDir(dir string) (ret []fs.FileInfo, err error) {
	var fpath = dir
	var file *os.File
	for len(fpath) > 0 {
		if file, err = os.Open(fpath); err == nil {
			defer file.Close()
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return
		}
		fpath = path.Dir(fpath)
	}
	if fpath == dir { // primary filesystem directory
		var fi fs.FileInfo
		if fi, err = file.Stat(); err != nil {
			return
		}
		if fi.IsDir() {
			return file.Readdir(-1)
		}
	}

	var dv interface{}
	if dv, err = diskcache.Get(fpath); err != nil {
		return
	}
	if err = diskcache.Set(fpath, dv); err != nil { // update expiration time
		return
	}

	var dpath string
	if fpath == dir {
		dpath = "/" // list root of disk
	} else {
		dpath = dir[len(fpath):]
	}
	switch disk := dv.(type) {
	case *DiskISO:
		var enc = charmap.Windows1251.NewEncoder()
		dpath, _ = enc.String(dpath)
		return disk.fs.ReadDir(dpath)
	}
	panic("not released disk type present")
}

// ScanDir returns file properties list for given file system directory, or directory in iso-disk.
func ScanDir(dir string, cg *CatGrp, prf *Profile) (ret []Pather, skip int, err error) {
	var files []fs.FileInfo
	if files, err = OpenDir(dir); err != nil && len(files) == 0 {
		return
	}

	var t1 = time.Now()

	var fgrp FileGroup
	for _, fi := range files {
		if fi == nil {
			continue
		}
		var fpath = path.Join(dir, fi.Name())
		if prf.IsHidden(fpath) {
			continue
		}
		var grp = GetFileGroup(fpath)
		if cg[grp] {
			var prop Pather
			if propcache.Has(fpath) {
				var pv, _ = propcache.Get(fpath)
				prop = pv.(Pather)
			} else {
				prop = MakeProp(fpath, fi)
				propcache.Set(fpath, prop)
			}
			ret = append(ret, prop)
		}
		*fgrp.Field(grp)++
	}
	skip = len(files) - len(ret)

	var latency = int(time.Until(t1) / 1000000)

	var puid, _ = PathCachePUID(dir)
	if err = DirCacheSet(&DirCacheItem{
		Puid: puid,
		DirProp: DirProp{
			Scan:    UnixJSNow(),
			FGrp:    fgrp,
			Latency: latency,
		},
	}); err != nil {
		return
	}

	return
}

// The End.
