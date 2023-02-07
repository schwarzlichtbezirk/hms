package hms

import (
	"fmt"
	"io/fs"
	"path"
	"time"
)

// ScanFileNameList returns file properties list for given list of
// full file system paths. File paths can be in different folders.
func ScanFileNameList(prf *Profile, session *Session, vpaths []string) (ret []any, lstp DirProp, err error) {
	var files = make([]fs.FileInfo, len(vpaths))
	for i, fpath := range vpaths {
		var fi, _ = StatFile(fpath)
		files[i] = fi
	}

	return ScanFileInfoList(prf, session, files, vpaths)
}

// ScanFileInfoList returns file properties list for given list of
// []fs.FileInfo and associated list of full file system paths.
// Elements of []fs.FileInfo list can be nil in case if file is
// unavailable, or if it categoty item.
func ScanFileInfoList(prf *Profile, session *Session, vfiles []fs.FileInfo, vpaths []string) (ret []any, lstp DirProp, err error) {
	var tscan = time.Now()

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

	// update dir cache
	var dpmap = map[Puid_t]DirProp{}
	var idds []Puid_t
	for _, puid := range vpuids {
		if dp, ok := dircache.Get(puid); ok {
			dpmap[puid] = dp
		} else { // get directories, ISO-files and playlists as folder properties
			idds = append(idds, puid)
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

	// format response
	for i, fpath := range vpaths {
		var puid = vpuids[i]
		var fp FileProp
		var fi = vfiles[i]
		if name, ok := CatNames[puid]; ok {
			fp.Name = name
		} else {
			fp.Name = path.Base(fpath)
		}
		fp.Type = prf.PathType(fpath, fi)
		if fi != nil {
			fp.Size = fi.Size()
			fp.Time = fi.ModTime()
		}
		var grp = prf.GetPathGroup(fpath, fi)
		*lstp.FGrp.Field(grp)++

		if dp, ok := dpmap[puid]; ok || fp.Type != FTfile {
			var dk DirKit
			dk.PUID = puid
			dk.Free = prf.PathAccess(fpath, false)
			dk.Shared = prf.IsShared(fpath)
			if fi != nil {
				_, dk.Static = fi.(*FileInfoISO)
			} else {
				dk.Static = true
			}
			dk.FileProp = fp
			dk.DirProp = dp
			if vfiles[i] == nil && dk.Type != FTctgr {
				dk.Latency = -1
			}
			ret = append(ret, &dk)
		} else {
			var fk FileKit
			fk.PUID = puid
			fk.Free = prf.PathAccess(fpath, false)
			fk.Shared = prf.IsShared(fpath)
			if fi != nil {
				_, fk.Static = fi.(*FileInfoISO)
			} else {
				fk.Static = true
			}
			fk.FileProp = fp
			if tp, ok := tilecache.Peek(puid); ok {
				fk.TileProp = *tp
			}
			ret = append(ret, &fk)
		}
	}

	lstp.Scan = tscan
	lstp.Latency = int(time.Since(tscan) / time.Millisecond)

	return
}

// ScanDir returns file properties list for given file system directory,
// or directory in iso-disk.
func ScanDir(prf *Profile, session *Session, dir string, isadmin bool) (ret []any, skipped int, err error) {
	var files []fs.FileInfo
	if files, err = ReadDir(dir); err != nil && len(files) == 0 {
		return
	}

	/////////////////////////////
	// define files to display //
	/////////////////////////////

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
		if !prf.PathAccess(fpath, isadmin) {
			continue
		}

		vfiles = append(vfiles, fi)
		vpaths = append(vpaths, fpath)
	}
	skipped = len(files) - len(vfiles)

	var dp DirProp
	if ret, dp, err = ScanFileInfoList(prf, session, vfiles, vpaths); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		var puid = PathStoreCache(session, dir)
		DirStoreSet(session, &DirStore{
			Puid: puid,
			Prop: dp,
		})
		return
	})

	return
}

// ScanCat returns file properties list where number of files
// of given category is more then given percent.
func ScanCat(prf *Profile, session *Session, puid Puid_t, cat string, percent float64) (ret []any, err error) {
	const categoryCond = "(%s)/(other+video+audio+image+books+texts+packs) > %f"
	var dss []DirStore
	if err = session.Where(fmt.Sprintf(categoryCond, cat, percent)).Find(&dss); err != nil {
		return
	}
	var newpuids []uint64
	var vpaths []string
	for _, ds := range dss {
		dircache.Set(ds.Puid, ds.Prop)
		if fpath, ok := pathcache.GetDir(ds.Puid); ok {
			vpaths = append(vpaths, fpath)
		} else {
			newpuids = append(newpuids, uint64(ds.Puid))
		}
	}
	if len(newpuids) > 0 {
		var nps []PathStore
		// get not cached paths from database
		if err = session.In("puid", newpuids).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			pathcache.Set(ps.Puid, ps.Path)
			vpaths = append(vpaths, ps.Path)
		}
	}

	var dp DirProp
	if ret, dp, err = ScanFileNameList(prf, session, vpaths); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, &DirStore{
			Puid: puid,
			Prop: dp,
		})
		return
	})

	return
}

// The End.
