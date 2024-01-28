package cmd

import (
	"fmt"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	srv "github.com/schwarzlichtbezirk/hms/server"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

const (
	cfgfile = "hms.yaml"
	prffile = "profiles.yaml"
	passlst = "passlist.yaml"

	dirfile = "hms-storage.sqlite"
	userlog = "hms-userlog.sqlite"
)

var (
	JoinPath = srv.JoinPath
	Cfg      = cfg.Cfg
	Log      = cfg.Log
)

// InitStorage inits database caches engine.
func InitStorage() (err error) {
	if srv.XormStorage, err = xorm.NewEngine(Cfg.XormDriverName, JoinPath(cfg.SqlPath, dirfile)); err != nil {
		return
	}
	srv.XormStorage.SetMapper(names.GonicMapper{})
	var xlb = cfg.XormLoggerBridge{
		Logger: Log,
	}
	xlb.ShowSQL(cfg.DevMode)
	srv.XormStorage.SetLogger(&xlb)

	_, err = srv.SqlSession(func(session *xorm.Session) (res any, err error) {
		if err = session.Sync(&srv.PathStore{}, &srv.DirStore{}, &srv.ExtStore{}, &srv.ExifStore{}, &srv.Id3Store{}); err != nil {
			return
		}

		// fill path_store & file_store with predefined items
		var ok bool
		if ok, err = session.IsTableEmpty(&srv.PathStore{}); err != nil {
			return
		}
		if ok {
			var ctgrpath = make([]srv.PathStore, srv.PUIDcache-1)
			for puid, path := range srv.CatKeyPath {
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				srv.PathCache.Set(puid, path)
			}
			for puid := srv.Puid_t(len(srv.CatKeyPath) + 1); puid < srv.PUIDcache; puid++ {
				var path = fmt.Sprintf("<reserved%d>", puid)
				ctgrpath[puid-1].Puid = puid
				ctgrpath[puid-1].Path = path
				srv.PathCache.Set(puid, path)
			}
			if _, err = session.Insert(&ctgrpath); err != nil {
				return
			}
		}
		return
	})
	return
}

// InitUserlog inits database user log engine.
func InitUserlog() (err error) {
	if srv.XormUserlog, err = xorm.NewEngine(Cfg.XormDriverName, JoinPath(cfg.SqlPath, userlog)); err != nil {
		return
	}
	srv.XormUserlog.SetMapper(names.GonicMapper{})
	srv.XormUserlog.ShowSQL(false)

	if err = srv.XormUserlog.Sync(&srv.AgentStore{}, &srv.OpenStore{}); err != nil {
		return
	}

	var uacount, _ = srv.XormUserlog.Count(&srv.AgentStore{})
	Log.Infof("user agent count %d items", uacount)
	var opencount, _ = srv.XormUserlog.Count(&srv.OpenStore{})
	Log.Infof("resources open count %d items", opencount)
	return
}

// LoadPathCache loads whole path table from database into cache.
func LoadPathCache() (err error) {
	var session = srv.XormStorage.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var chunk []srv.PathStore
		if err = session.Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ps := range chunk {
			srv.PathCache.Set(ps.Puid, ps.Path)
		}
		if limit > len(chunk) {
			break
		}
	}

	Log.Infof("loaded %d items into path cache", srv.PathCache.Len())
	return
}

// LoadGpsCache loads all items with GPS information from EXIF table of storage into cache.
func LoadGpsCache() (err error) {
	var session = srv.XormStorage.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var chunk []srv.ExifStore
		if err = session.Where("latitude != 0").Cols("puid", "datetime", "latitude", "longitude", "altitude").Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ec := range chunk {
			var gi srv.GpsInfo
			gi.FromProp(&ec.Prop)
			srv.GpsCache.Poke(ec.Puid, gi)
		}
		if limit > len(chunk) {
			break
		}
	}

	Log.Infof("loaded %d items into GPS cache", srv.GpsCache.Len())
	return
}
