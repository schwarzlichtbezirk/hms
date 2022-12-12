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

	var dv any
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

	var dv any
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

func ScanFileNameList(prf *Profile, vpaths []string) (ret []Pather, err error) {
	var files = make([]fs.FileInfo, len(vpaths))
	for i, fpath := range vpaths {
		var fi, _ = StatFile(fpath)
		files[i] = fi
	}

	return ScanFileInfoList(prf, files, vpaths)
}

func ScanFileInfoList(prf *Profile, vfiles []fs.FileInfo, vpaths []string) (ret []Pather, err error) {
	var session = xormEngine.NewSession()
	defer session.Close()

	var dpaths []string // database paths
	for _, fpath := range vpaths {
		if _, ok := pathcache.GetRev(fpath); !ok {
			dpaths = append(dpaths, fpath)
		}
	}

	if len(dpaths) > 0 {
		var nps []PathStore
		// get not cached paths from database
		nps = nil
		if err = session.In("path", dpaths).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			pathcache.Set(ps.Puid, ps.Path)
		}
		// insert new paths into database
		nps = nil
		var npaths []string // new paths
		for _, fpath := range dpaths {
			if _, ok := pathcache.GetRev(fpath); !ok {
				nps = append(nps, PathStore{
					Path: fpath,
				})
				npaths = append(npaths, fpath)
			}
		}
		if _, err = session.Insert(nps); err != nil {
			return
		}
		// get PUIDs of inserted paths
		nps = nil
		if err = session.In("path", npaths).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			pathcache.Set(ps.Puid, ps.Path)
		}
	}

	// make vpuids array
	var vpuids []Puid_t // verified PUIDs
	for _, fpath := range vpaths {
		var puid, _ = pathcache.GetRev(fpath)
		vpuids = append(vpuids, puid)
	}

	////////////////////////////
	// define files to upsert //
	////////////////////////////

	var vfs []FileStore // verified file store list
	var oldfs, newfs, updfs []FileStore
	if err = session.In("puid", vpuids).Find(&oldfs); err != nil {
		return
	}
	for i, fpath := range vpaths {
		var fs FileStore
		var puid = vpuids[i]
		var fi = vfiles[i]
		var found = false
		for _, v := range oldfs {
			if v.Puid == puid {
				fs = v
				if fi != nil {
					var sizeval = fi.Size()
					var timeval = UnixJS(fi.ModTime())
					var typeval = prf.PathType(fpath, fi)
					if fs.Prop.TypeVal != typeval || fs.Prop.SizeVal != sizeval || fs.Prop.TimeVal != timeval {
						fs.Prop.TypeVal = typeval
						fs.Prop.SizeVal = sizeval
						fs.Prop.TimeVal = timeval
						updfs = append(updfs, fs)
					}
				}
				found = true
				break
			}
		}
		if !found {
			fs.Puid = puid
			fs.Prop.PathProp = PathProp{
				NameVal: path.Base(fpath),
				TypeVal: prf.PathType(fpath, fi),
			}
			if fi != nil {
				fs.Prop.SizeVal = fi.Size()
				fs.Prop.TimeVal = UnixJS(fi.ModTime())
			}
			newfs = append(newfs, fs)
		}
		vfs = append(vfs, fs)
	}

	if len(newfs) > 0 || len(updfs) > 0 {
		go xormEngine.Transaction(func(session *Session) (res any, err error) {
			// insert new items
			if len(newfs) > 0 {
				if _, err = session.Insert(newfs); err != nil {
					return
				}
			}

			// update changed items
			for _, fs := range updfs {
				if _, err = session.ID(fs.Puid).Cols("type", "size", "time").Update(&fs); err != nil {
					return
				}
			}
			return
		})
	}

	//////////////////////
	// update dir cache //
	//////////////////////

	var dpmap = map[Puid_t]DirProp{}
	var idds []Puid_t
	for _, fs := range vfs {
		if fs.Prop.TypeVal != FTfile {
			if dp, ok := dircache.Get(fs.Puid); ok {
				dpmap[fs.Puid] = dp
			} else {
				idds = append(idds, fs.Puid)
			}
		}
	}
	if len(idds) > 0 {
		var dss []DirStore
		if err = session.In("puid", idds).Find(&dss); err != nil {
			return
		}
		for _, ds := range dss {
			dpmap[ds.Puid] = ds.Prop
			dircache.Set(ds.Puid, ds.Prop)
		}
	}

	/////////////////////
	// format response //
	/////////////////////

	for i, fs := range vfs {
		if fs.Prop.TypeVal != FTfile {
			var dk DirKit
			dk.FileProp = fs.Prop
			dk.PUIDVal = fs.Puid
			if dp, ok := dpmap[fs.Puid]; ok {
				dk.DirProp = dp
			}
			if vfiles[i] == nil && dk.TypeVal != FTctgr {
				dk.Latency = -1
			}
			ret = append(ret, &dk)
		} else {
			var fk FileKit
			fk.FileProp = fs.Prop
			fk.PUIDVal = fs.Puid
			//fk.TmbProp.Setup(fpath)
			ret = append(ret, &fk)
		}
	}

	return
}

// ScanDir returns file properties list for given file system directory, or directory in iso-disk.
func ScanDir(prf *Profile, dir string, cg *CatGrp) (ret []Pather, skip int, err error) {
	var tscan = time.Now()

	var files []fs.FileInfo
	if files, err = OpenDir(dir); err != nil && len(files) == 0 {
		return
	}

	/////////////////////////////
	// define files to display //
	/////////////////////////////

	var fgrp FileGroup
	var vfiles []fs.FileInfo // verified file infos
	var vpaths []string      // verified paths
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

	if ret, err = ScanFileInfoList(prf, vfiles, vpaths); err != nil {
		return
	}

	var latency = int(time.Since(tscan) / time.Millisecond)

	go xormEngine.Transaction(func(session *Session) (res any, err error) {
		var puid = PathStoreCache(session, dir)
		DirStoreSet(session, &DirStore{
			Puid: puid,
			Prop: DirProp{
				Scan:    UnixJS(tscan),
				FGrp:    fgrp,
				Latency: latency,
			},
		})
		return
	})

	return
}

// The End.
