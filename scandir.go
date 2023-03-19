package hms

import (
	"fmt"
	"io/fs"
	"path"
	"time"
)

// ScanFileNameList returns file properties list for given list of
// full file system paths. File paths can be in different folders.
func ScanFileNameList(prf *Profile, session *Session, vpaths []DiskPath) (ret []any, lstp DirProp, err error) {
	var files = make([]fs.FileInfo, len(vpaths))
	for i, dp := range vpaths {
		var fi, _ = StatFile(dp.Path)
		files[i] = fi
	}

	return ScanFileInfoList(prf, session, files, vpaths)
}

// ScanFileInfoList returns file properties list for given list of
// []fs.FileInfo and associated list of full file system paths.
// Elements of []fs.FileInfo list can be nil in case if file is
// unavailable, or if it categoty item.
func ScanFileInfoList(prf *Profile, session *Session, vfiles []fs.FileInfo, vpaths []DiskPath) (ret []any, lstp DirProp, err error) {
	var tscan = time.Now()

	var dpaths []string // database paths
	for _, dp := range vpaths {
		if _, ok := pathcache.GetRev(dp.Path); !ok {
			dpaths = append(dpaths, dp.Path)
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
	for _, dp := range vpaths {
		var puid, _ = pathcache.GetRev(dp.Path)
		vpuids = append(vpuids, puid)
	}

	// get directories, ISO-files and playlists as folder properties
	var idds []Puid_t
	for _, puid := range vpuids {
		if !dircache.Has(puid) {
			idds = append(idds, puid)
		}
	}
	if len(idds) > 0 {
		var dss []DirStore
		if err = session.In("puid", idds).Find(&dss); err != nil {
			return
		}
		for _, ds := range dss {
			dircache.Poke(ds.Puid, ds.Prop)
		}
	}

	// format response
	for i, dp := range vpaths {
		var fpath = dp.Path
		var puid = vpuids[i]
		var fi = vfiles[i]
		var pp = PuidProp{
			PUID:   puid,
			Free:   prf.PathAccess(fpath, false),
			Shared: prf.IsShared(fpath),
		}
		if fi != nil {
			_, pp.Static = fi.(*FileInfoISO)
		} else {
			pp.Static = true
		}
		var fp FileProp
		fp.Name = dp.Name
		fp.Type = prf.PathType(fpath, fi)
		if fi != nil {
			fp.Size = fi.Size()
			fp.Time = fi.ModTime()
		}
		var grp = prf.GetPathGroup(fpath, fi)
		*lstp.FGrp.Field(grp)++

		if dp, ok := dircache.Peek(puid); ok || fp.Type != FTfile {
			var dk = DirKit{
				PuidProp: pp,
				FileProp: fp,
				DirProp:  dp,
			}
			if vfiles[i] == nil && dk.Type != FTctgr {
				dk.Latency = -1
			}
			ret = append(ret, &dk)
		} else {
			var fk = FileKit{
				PuidProp: pp,
				FileProp: fp,
			}
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
	var vpaths []DiskPath    // verified paths
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
		vpaths = append(vpaths, DiskPath{fpath, fi.Name()})
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
	var vpaths []DiskPath
	for _, ds := range dss {
		dircache.Set(ds.Puid, ds.Prop)
		if fpath, ok := pathcache.GetDir(ds.Puid); ok {
			vpaths = append(vpaths, MakeFilePath(fpath))
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
			vpaths = append(vpaths, MakeFilePath(ps.Path))
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
