package hms

import (
	"encoding/xml"
	"io/fs"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rwcarlsen/goexif/exif"
)

/*

#include <stdio.h>
#include <stdlib.h>

extern double haversine(double lat1, double lon1, double lat2, double lon2);

*/
import "C"

type Point struct {
	Latitude  float64 `json:"lat" yaml:"lat" xml:"lat,attr"`
	Longitude float64 `json:"lon" yaml:"lon" xml:"lon,attr"`
}

type Shape string

const (
	Circle    Shape = "circle"
	Polygon   Shape = "polygon"
	Rectangle Shape = "rectangle"
)

// MapPath describes any map path that can contains a points.
type MapPath struct {
	Shape  Shape   `json:"shape" yaml:"shape" xml:"shape"`
	Eject  bool    `json:"eject" yaml:"eject" xml:"eject"`
	Radius float64 `json:"radius,omitempty" yaml:"radius,omitempty" xml:"radius,omitempty"`
	Coord  []Point `json:"coord" yaml:"coord,flow" xml:"coord>point"`
}

func (mp *MapPath) Contains(lat, lon float64) bool {
	switch mp.Shape {
	case Circle:
		var d = float64(C.haversine(C.double(mp.Coord[0].Latitude), C.double(mp.Coord[0].Longitude), C.double(lat), C.double(lon)))
		return d <= mp.Radius
	default:
		panic(ErrShapeBad)
	}
}

// APIHANDLER
func SpiGpsRange(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Paths []MapPath `json:"paths" yaml:"paths" xml:"paths>path"`
		Limit int       `json:"limit,omitempty" yaml:"limit,omitempty" xml:"limit,omitempty"`
		Time1 time.Time `json:"time1,omitempty" yaml:"time1,omitempty" xml:"time1,omitempty"`
		Time2 time.Time `json:"time2,omitempty" yaml:"time2,omitempty" xml:"time2,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []any `json:"list" yaml:"list" xml:"list>prop"`

		HasHome bool `json:"hashome" yaml:"hashome" xml:"hashome,attr"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_gpsrange_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_gpsrange_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_gpsrange_noacc, ErrNoAcc)
		return
	}

	for _, mp := range arg.Paths {
		switch mp.Shape {
		case Circle:
			if len(mp.Coord) != 1 {
				Ret400(c, SEC_gpsrange_shpcirc, ErrShapeCirc)
				return
			}
		case Polygon:
			if len(mp.Coord) < 3 {
				Ret400(c, SEC_gpsrange_shppoly, ErrShapePoly)
				return
			}
		case Rectangle:
			if len(mp.Coord) != 4 {
				Ret400(c, SEC_gpsrange_shprect, ErrShapeRect)
				return
			}
		default:
			Ret400(c, SEC_gpsrange_shpbad, ErrShapeBad)
			return
		}
	}
	if arg.Limit < 0 {
		arg.Limit = Cfg.RangeSearchLimit
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	if uid == aid {
		ret.HasHome = true
	} else if acc.IsShared(CPhome) {
		for _, fpath := range CatKeyPath {
			if fpath == CPhome {
				continue
			}
			if acc.IsShared(fpath) {
				ret.HasHome = true
				break
			}
		}
	}

	var vfiles []fs.FileInfo // verified file infos
	var vpaths []DiskPath    // verified paths
	GpsCache.Range(func(puid Puid_t, gps GpsInfo) bool {
		var inc bool
		for _, mp := range arg.Paths {
			if !mp.Contains(gps.Latitude, gps.Longitude) {
				continue
			}
			if !arg.Time1.IsZero() && gps.DateTime.Before(arg.Time1) {
				continue
			}
			if !arg.Time2.IsZero() && gps.DateTime.After(arg.Time2) {
				continue
			}
			inc = !mp.Eject
		}
		if inc {
			var fpath, _ = PathStorePath(session, puid)
			if !Hidden.Fits(fpath) && acc.PathAccess(fpath, uid == aid) {
				if fi, _ := JP.Stat(fpath); fi != nil {
					vfiles = append(vfiles, fi)
					vpaths = append(vpaths, MakeFilePath(fpath))
				}
			}
		}
		return arg.Limit == 0 || len(ret.List) < arg.Limit
	})
	if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths, true); err != nil {
		Ret500(c, SEC_gpsrange_list, err)
		return
	}

	RetOk(c, ret)
}

// APIHANDLER
func SpiGpsScan(c *gin.Context) {
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		List []Puid_t `json:"list" yaml:"list" xml:"list>puid" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Store[GpsInfo] `json:"list" yaml:"list" xml:"list>tile"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_gpsscan_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid ID_t
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_gpsscan_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_gpsscan_noacc, ErrNoAcc)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var ests []ExifStore
	for _, puid := range arg.List {
		var puid = puid // localize
		if syspath, ok := PathStorePath(session, puid); ok {
			if !acc.PathAccess(syspath, uid == aid) {
				continue
			}
			if val, ok := GpsCache.Peek(puid); ok {
				var gst = Store[GpsInfo]{
					Puid: puid,
					Prop: val,
				}
				ret.List = append(ret.List, gst)
			} else {
				// try to get from database
				var err error
				var est ExifStore
				if ok, err = session.ID(puid).Get(&est); err != nil {
					continue
				}
				if ok {
					if est.Prop.IsZero() {
						continue
					}
				} else {
					// try to extract from file
					func() (err error) {
						var file fs.File
						if file, err = JP.Open(syspath); err != nil {
							return
						}
						defer file.Close()

						var x *exif.Exif
						if x, err = exif.Decode(file); err != nil {
							return
						}

						est.Prop.Setup(x)
						return
					}()
					if err != nil || est.Prop.IsZero() {
						continue
					}
					// prepare to set to database
					ests = append(ests, est)
				}
				if est.Prop.Latitude != 0 || est.Prop.Longitude != 0 {
					var gst Store[GpsInfo]
					gst.Puid = puid
					gst.Prop.FromProp(&est.Prop)
					ret.List = append(ret.List, gst)
					// set to GPS cache
					GpsCache.Poke(puid, gst.Prop)
				}
			}
		}
	}

	if len(ests) > 0 {
		go SqlSession(func(session *Session) (res any, err error) {
			_, err = session.Insert(&ests)
			return
		})
	}

	RetOk(c, ret)
}

// The End.
