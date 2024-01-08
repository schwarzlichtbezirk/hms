package hms

import (
	"fmt"
	"io/fs"
	"time"

	jnt "github.com/schwarzlichtbezirk/joint"
)

type SubPool struct {
	*jnt.JointPool
	Dir string
}

func (sp *SubPool) Open(fpath string) (f fs.File, err error) {
	var fullpath = jnt.JoinFast(sp.Dir, fpath)
	return sp.JointPool.Open(fullpath)
}

func (sp *SubPool) Stat(fpath string) (fi fs.FileInfo, err error) {
	var fullpath = jnt.JoinFast(sp.Dir, fpath)
	return sp.JointPool.Stat(fullpath)
}

func (sp *SubPool) ReadDir(fpath string) (ret []fs.DirEntry, err error) {
	var fullpath = jnt.JoinFast(sp.Dir, fpath)
	return sp.JointPool.ReadDir(fullpath)
}

func (sp *SubPool) Sub(dir string) (fs.FS, error) {
	var fulldir = jnt.JoinFast(sp.Dir, dir)
	var fi, err = sp.JointPool.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() && jnt.IsTypeIso(dir) {
		return nil, fs.ErrNotExist
	}
	return &SubPool{
		JointPool: sp.JointPool,
		Dir:       fulldir,
	}, nil
}

// var JP = jnt.NewSubPool(jnt.NewJointPool(), "")
var JP = SubPool{jnt.NewJointPool(), ""}

type RFile = jnt.RFile

func OpenFile(fpath string) (file RFile, err error) {
	var f fs.File
	if f, err = JP.Open(fpath); err != nil {
		return
	}
	file = f.(RFile)
	return
}

// IsStatic returns whether file info refers to content
// that can not be modified or moved.
func IsStatic(fi fs.FileInfo) (static bool) {
	if static = fi == nil; static {
		return
	}
	if _, static = fi.(jnt.IsoFileInfo); static {
		return
	}
	if _, static = fi.(jnt.DavFileInfo); static {
		return
	}
	if _, static = fi.(jnt.FtpFileInfo); static {
		return
	}
	if sys := fi.Sys(); sys != nil {
		if _, static = sys.(*jnt.SftpFileStat); static {
			return
		}
	}
	return
}

// ScanFileNameList returns file properties list for given list of
// full file system paths. File paths can be in different folders.
func ScanFileNameList(prf *Profile, session *Session, vpaths []DiskPath, scanembed bool) (ret []any, lstp DirProp, err error) {
	var files = make([]fs.FileInfo, len(vpaths))
	for i, dp := range vpaths {
		fi, _ := JP.Stat(dp.Path)
		files[i] = fi
	}

	return ScanFileInfoList(prf, session, files, vpaths, scanembed)
}

// ScanFileInfoList returns file properties list for given list of
// []fs.FileInfo and associated list of full file system paths.
// Elements of []fs.FileInfo list can be nil in case if file is
// unavailable, or if it categoty item.
func ScanFileInfoList(prf *Profile, session *Session, vfiles []fs.FileInfo, vpaths []DiskPath, scanembed bool) (ret []any, lstp DirProp, err error) {
	var tscan = time.Now()

	var vpuids = make([]Puid_t, len(vpaths)) // verified PUIDs

	var npaths = make([]string, 0, len(vpaths)) // new paths
	var nps = make([]PathStore, 0, len(vpaths))
	var pm = map[string]struct{}{}
	for i, dp := range vpaths {
		if puid, ok := PathStorePUID(session, dp.Path); ok {
			vpuids[i] = puid
		} else {
			npaths = append(npaths, dp.Path)
			if _, ok := pm[dp.Path]; !ok {
				nps = append(nps, PathStore{
					Path: dp.Path,
				})
				pm[dp.Path] = struct{}{}
			}
		}
	}
	if len(nps) > 0 {
		// insert new paths into database
		if _, err = session.Insert(&nps); err != nil {
			return
		}
		// get PUIDs of inserted paths
		nps = nil
		if err = session.In("path", npaths).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			PathCache.Set(ps.Puid, ps.Path)
		}
		// get remained vpuids
		for i, dp := range vpaths {
			if vpuids[i] == 0 {
				var puid, _ = PathCache.GetRev(dp.Path)
				vpuids[i] = puid
			}
		}
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
	var extmap = map[Puid_t]ExtProp{}
	var ess []ExtStore
	var epuids = make([]Puid_t, 0, len(vpuids)) // ext
	for i, puid := range vpuids {
		if xp, ok := extcache.Peek(puid); ok {
			extmap[puid] = xp
		} else {
			var ext = GetFileExt(vpaths[i].Path)
			if IsTypeEXIF(ext) || IsTypeDecoded(ext) || IsTypeID3(ext) {
				epuids = append(epuids, puid)
			} else {
				extmap[puid] = ExtProp{
					Tags: TagDis,
					ETmb: MimeDis,
				}
			}
		}
	}
	if len(epuids) > 0 {
		if err = session.In("puid", epuids).Find(&ess); err != nil {
			return
		}
		for _, es := range ess {
			extcache.Poke(es.Puid, es.Prop)
			extmap[es.Puid] = es.Prop
		}
	}
	// scan not cached
	if scanembed {
		for _, puid := range epuids {
			if _, ok := extmap[puid]; !ok {
				ImgScanner.AddTags(puid)
			}
		}
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
			Static: IsStatic(fi),
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
				fk.TileProp = tp
			}
			if xp, ok := extmap[puid]; ok {
				fk.ExtProp = xp
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
func ScanDir(prf *Profile, session *Session, dir string, isadmin bool, scanembed bool) (ret []any, skipped int, err error) {
	var files []fs.DirEntry
	if files, err = JP.ReadDir(dir); err != nil && len(files) == 0 {
		return
	}

	/////////////////////////////
	// define files to display //
	/////////////////////////////

	var vfiles = make([]fs.FileInfo, 0, len(files)) // verified file infos
	var vpaths = make([]DiskPath, 0, len(files))    // verified paths
	for _, de := range files {
		var fi fs.FileInfo
		if fi, err = de.Info(); err != nil {
			continue
		}
		var fpath = JoinPath(dir, fi.Name())
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
	if ret, dp, err = ScanFileInfoList(prf, session, vfiles, vpaths, scanembed); err != nil {
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
func ScanCat(prf *Profile, session *Session, puid Puid_t, cat string, percent float64, scanembed bool) (ret []any, err error) {
	const categoryCond = "(%s)/(other+video+audio+image+books+texts+packs) > %f"
	var dss []DirStore
	if err = session.Where(fmt.Sprintf(categoryCond, cat, percent)).Find(&dss); err != nil {
		return
	}
	var newpuids []uint64
	var vpaths []DiskPath
	for _, ds := range dss {
		if fpath, ok := PathStorePath(session, ds.Puid); ok {
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
	if ret, dp, err = ScanFileNameList(prf, session, vpaths, scanembed); err != nil {
		return
	}

	go SqlSession(func(session *Session) (res any, err error) {
		DirStoreSet(session, puid, dp)
		return
	})

	return
}

// The End.
