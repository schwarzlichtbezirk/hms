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
		var grp = prf.GetPathGroup(fpath, fi)
		*lstp.FGrp.Field(grp)++
		var found = false
		for _, v := range oldfs {
			if v.Puid == puid {
				fs = v
				if fi != nil {
					var sizeval = fi.Size()
					var timeval = UnixJS(fi.ModTime())
					var typeval = prf.PathType(fpath, fi)
					if fs.Prop.Type != typeval || fs.Prop.Size != sizeval || fs.Prop.Time != timeval {
						fs.Prop.Type = typeval
						fs.Prop.Size = sizeval
						fs.Prop.Time = timeval
						updfs = append(updfs, fs)
					}
				}
				found = true
				break
			}
		}
		if !found {
			fs.Puid = puid
			fs.Prop.Name = path.Base(fpath)
			fs.Prop.Type = prf.PathType(fpath, fi)
			if fi != nil {
				fs.Prop.Size = fi.Size()
				fs.Prop.Time = UnixJS(fi.ModTime())
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
		if fs.Prop.Type != FTfile {
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
		if fs.Prop.Type != FTfile {
			var dk DirKit
			dk.FileProp = fs.Prop
			dk.PUID = fs.Puid
			if dp, ok := dpmap[fs.Puid]; ok {
				dk.DirProp = dp
			}
			if vfiles[i] == nil && dk.Type != FTctgr {
				dk.Latency = -1
			}
			ret = append(ret, &dk)
		} else {
			var fk FileKit
			fk.FileProp = fs.Prop
			fk.PUID = fs.Puid
			fk.TileProp, _ = tilecache.Peek(fs.Puid)
			ret = append(ret, &fk)
		}
	}

	lstp.Scan = UnixJS(tscan)
	lstp.Latency = int(time.Since(tscan) / time.Millisecond)

	return
}

// ScanDir returns file properties list for given file system directory,
// or directory in iso-disk.
func ScanDir(prf *Profile, session *Session, dir string, cg *CatGrp) (ret []any, skip int, err error) {
	var files []fs.FileInfo
	if files, err = OpenDir(dir); err != nil && len(files) == 0 {
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
		var grp = prf.GetPathGroup(fpath, fi)
		if !cg[grp] {
			continue
		}

		vfiles = append(vfiles, fi)
		vpaths = append(vpaths, fpath)
	}
	skip = len(files) - len(vfiles)

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
