package hms

import (
	"fmt"
	"io/fs"
	"time"

	jnt "github.com/schwarzlichtbezirk/hms/joint"
)

// ScanFileNameList returns file properties list for given list of
// full file system paths. File paths can be in different folders.
func ScanFileNameList(prf *Profile, session *Session, vpaths []DiskPath) (ret []any, lstp DirProp, err error) {
	var files = make([]fs.FileInfo, len(vpaths))
	for i, dp := range vpaths {
		fi, _ := jnt.StatFile(dp.Path)
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

	var dpaths = make([]string, 0, len(vpaths)) // database paths
	for _, dp := range vpaths {
		if _, ok := PathCache.GetRev(dp.Path); !ok {
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
			PathCache.Set(ps.Puid, ps.Path)
		}
		// insert new paths into database
		nps = make([]PathStore, 0, len(dpaths))
		var npaths = make([]string, 0, len(dpaths)) // new paths
		for _, fpath := range dpaths {
			if _, ok := PathCache.GetRev(fpath); !ok {
				nps = append(nps, PathStore{
					Path: fpath,
				})
				npaths = append(npaths, fpath)
			}
		}
		if _, err = session.Insert(&nps); err != nil {
			return
		}
		// get PUIDs of inserted paths
		nps = nil
		if err = session.In("path", npaths).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			if ps.Puid != 0 {
				PathCache.Set(ps.Puid, ps.Path)
			}
		}
	}

	// make vpuids array
	var vpuids = make([]Puid_t, len(vpaths)) // verified PUIDs
	for i, dp := range vpaths {
		var puid, _ = PathCache.GetRev(dp.Path)
		vpuids[i] = puid
	}

	// get directories, ISO-files and playlists as folder properties
	var dss []DirStore
	if err = session.In("puid", vpuids).Find(&dss); err != nil {
		return
	}
	var dirmap = map[Puid_t]DirProp{}
	for _, ds := range dss {
		dirmap[ds.Puid] = ds.Prop
	}

	// get extension info
	var ess []ExtStore
	var epuids = make([]Puid_t, len(vpaths)) // ext
	for i, dp := range vpaths {
		var ext = GetFileExt(dp.Path)
		if IsTypeEXIF(ext) || IsTypeDecoded(ext) || IsTypeID3(ext) {
			epuids = append(epuids, vpuids[i])
		}
	}
	if err = session.In("puid", epuids).Find(&ess); err != nil {
		return
	}
	var extmap = map[Puid_t]ExtProp{}
	for _, es := range ess {
		extmap[es.Puid] = es.Prop
	}

	// format response
	ret = make([]any, len(vpaths))
	for i, dp := range vpaths {
		var fpath = dp.Path
		var puid = vpuids[i]
		var fi = vfiles[i]
		var pp = PuidProp{
			PUID:   puid,
			Free:   prf.PathAccess(fpath, false),
			Shared: prf.IsShared(fpath),
			Static: jnt.IsStatic(fi),
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

		if dp, ok := dirmap[puid]; ok || fp.Type != FTfile {
			var dk = DirKit{
				PuidProp: pp,
				FileProp: fp,
				DirProp:  dp,
			}
			if vfiles[i] == nil && dk.Type != FTctgr {
				dk.Latency = -1
			}
			ret[i] = &dk
		} else {
			var fk = FileKit{
				PuidProp: pp,
				FileProp: fp,
			}
			if tp, ok := tilecache.Peek(puid); ok {
				fk.TileProp = *tp
			}
			if ep, ok := extmap[puid]; ok {
				fk.ExtProp = ep
			}
			ret[i] = &fk
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
	if files, err = jnt.ReadDir(dir); err != nil && len(files) == 0 {
		return
	}

	/////////////////////////////
	// define files to display //
	/////////////////////////////

	var vfiles = make([]fs.FileInfo, 0, len(files)) // verified file infos
	var vpaths = make([]DiskPath, 0, len(files))    // verified paths
	for _, fi := range files {
		if fi == nil {
			continue
		}
		var fpath = JoinFast(dir, fi.Name())
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
		DirStoreSet(session, puid, dp)
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
		if fpath, ok := PathCache.GetDir(ds.Puid); ok {
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
			PathCache.Set(ps.Puid, ps.Path)
			vpaths = append(vpaths, MakeFilePath(ps.Path))
		}
	}

	var dp DirProp
	if ret, dp, err = ScanFileNameList(prf, session, vpaths); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, puid, dp)
		return
	})

	return
}

// The End.
