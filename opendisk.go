package hms

import (
	"database/sql"
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

const sqlUpserPath = `
INSERT INTO path_cache (path,type,size,time) VALUES (?,?,?,?)
  ON CONFLICT(path) DO UPDATE SET type=?,size=?,time=?`

// ScanDir returns file properties list for given file system directory, or directory in iso-disk.
func ScanDir(dir string, cg *CatGrp, prf *Profile) (ret []Pather, skip int, err error) {
	var files []fs.FileInfo
	if files, err = OpenDir(dir); err != nil && len(files) == 0 {
		return
	}

	// start of scanning
	var t1 = time.Now()

	/////////////////////////////
	// define files to display //
	/////////////////////////////

	var fgrp FileGroup
	var vfiles []fs.FileInfo
	var vpaths []string
	for _, fi := range files {
		if fi == nil {
			continue
		}
		var fpath = path.Join(dir, fi.Name())
		if prf.IsHidden(fpath) {
			continue
		}
		var grp = GetFileGroup(fpath)
		if fi.IsDir() {
			grp = FGdir
		}
		if !cg[grp] {
			continue
		}
		*fgrp.Field(grp)++
		vfiles = append(vfiles, fi)
		vpaths = append(vpaths, fpath)
	}
	skip = len(files) - len(vfiles)

	////////////////////////////
	// define items to upsert //
	////////////////////////////

	var vpci []PathCacheItem
	if err = xormEngine.In("path", vpaths).Find(&vpci); err != nil {
		return
	}
	var inspci, updpci, constpci []*PathCacheItem
	for i, fi := range vfiles {
		var fpath = vpaths[i]
		var ins = true
		for _, pci := range vpci {
			if pci.Path == fpath {
				if pci.IsDiff(fi) {
					if fi.IsDir() {
						if prf.IsRoot(fpath) {
							pci.Type = FTdrv
						} else {
							pci.Type = FTdir
						}
					} else {
						pci.Type = FTfile
					}
					pci.Setup(fi)
					updpci = append(updpci, &pci)
				} else {
					constpci = append(constpci, &pci)
				}
				ins = false
				break
			}
		}
		if ins {
			var pci PathCacheItem
			pci.Path = fpath
			if fi.IsDir() {
				if prf.IsRoot(fpath) {
					pci.Type = FTdrv
				} else {
					pci.Type = FTdir
				}
			} else {
				pci.Type = FTfile
			}
			pci.PathInfo.Setup(fi)
			inspci = append(inspci, &pci)
		}
	}

	// insert new items
	if len(inspci) > 0 {
		for _, pci := range inspci {
			if _, err = xormEngine.InsertOne(pci); err != nil {
				return
			}
		}
	}

	// update changed items
	if len(updpci) > 0 {
		for _, pci := range updpci {
			if _, err = xormEngine.ID(pci.Puid).Cols("type", "size", "time").Update(pci); err != nil {
				return
			}
		}
	}

	_ = constpci // nothing to do with unchanged items

	///////////////////////
	// get PUIDs for all //
	///////////////////////

	var puidmap = map[string]Puid_t{}
	for _, pci := range vpci {
		puidmap[pci.Path] = pci.Puid
	}
	for _, pci := range inspci {
		puidmap[pci.Path] = pci.Puid
	}

	/////////////////////
	// format response //
	/////////////////////

	for i, fi := range vfiles {
		var fpath = vpaths[i]
		if fi.IsDir() {
			var dk DirKit
			dk.NameVal = PathBase(fpath)
			dk.TypeVal = FTdir
			if prf.IsRoot(fpath) {
				dk.TypeVal = FTdrv
			}
			dk.PUIDVal = puidmap[fpath]
			if dp, ok := DirCacheGet(dk.PUIDVal); ok {
				dk.DirProp = dp
			}
			ret = append(ret, &dk)
		} else {
			var fk FileKit
			fk.FileProp.Setup(fi)
			fk.PUIDVal = puidmap[fpath]
			fk.TmbProp.Setup(fpath)
			ret = append(ret, &fk)
		}
	}

	// end of scanning
	var latency = int(time.Since(t1) / time.Millisecond)

	go func() {
		var fi fs.FileInfo
		if fi, err = StatFile(dir); err != nil {
			return
		}

		var pci PathCacheItem
		pci.Path = dir
		if prf.IsRoot(dir) {
			pci.Type = FTdrv
		} else {
			pci.Type = FTdir
		}
		pci.PathInfo.Setup(fi)

		var res sql.Result
		res, err = xormEngine.Exec(sqlUpserPath,
			pci.Path, pci.Type, pci.Size, pci.Time,
			pci.Type, pci.Size, pci.Time)
		var ins, _ = res.LastInsertId()
		var aff, _ = res.RowsAffected()
		_ = aff

		var puid = Puid_t(ins)
		if puid == 0 {
			puid, _ = PathCachePUID(dir)
		}
		var dci = DirCacheItem{
			Puid: puid,
			DirProp: DirProp{
				Scan:    UnixJSNow(),
				FGrp:    fgrp,
				Latency: latency,
			},
		}
		if affected, _ := xormEngine.InsertOne(&dci); affected == 0 {
			_, err = xormEngine.ID(puid).AllCols().Omit("puid").Update(&dci)
		}
	}()

	return
}

// The End.
