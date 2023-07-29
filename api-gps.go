package hms

import (
	"encoding/xml"
	"io/fs"
	"math"
	"net/http"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	. "github.com/schwarzlichtbezirk/hms/config"
	. "github.com/schwarzlichtbezirk/hms/joint"
)

// Haversine uses formula to calculate the great-circle distance between
// two points – that is, the shortest distance over the earth’s surface –
// giving an ‘as-the-crow-flies’ distance between the points (ignoring
// any hills they fly over, of course!).
//
// See https://www.movable-type.co.uk/scripts/latlong.html
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // metres
	const πrad = math.Pi / 180
	var (
		φ1 = lat1 * πrad // φ, λ in radians
		φ2 = lat2 * πrad
		Δφ = (lat2 - lat1) * πrad
		Δλ = (lon2 - lon1) * πrad
		a  = math.Sin(Δφ/2)*math.Sin(Δφ/2) +
			math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
		c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
		d = R * c // in metres
	)
	return d
}

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
		var d = Haversine(mp.Coord[0].Latitude, mp.Coord[0].Longitude, lat, lon)
		return d <= mp.Radius
	default:
		panic(ErrShapeBad)
	}
}

// APIHANDLER
func gpsrangeAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
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
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}
	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECgpsrangenoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	for _, mp := range arg.Paths {
		switch mp.Shape {
		case Circle:
			if len(mp.Coord) != 1 {
				WriteError400(w, r, ErrShapeCirc, AECgpsrangeshpcirc)
				return
			}
		case Polygon:
			if len(mp.Coord) < 3 {
				WriteError400(w, r, ErrShapePoly, AECgpsrangeshppoly)
				return
			}
		case Rectangle:
			if len(mp.Coord) != 4 {
				WriteError400(w, r, ErrShapeRect, AECgpsrangeshprect)
				return
			}
		default:
			WriteError400(w, r, ErrShapeBad, AECgpsrangeshpbad)
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
	gpscache.Range(func(puid Puid_t, gps GpsInfo) bool {
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
			var fpath, _ = pathcache.GetDir(puid)
			if !acc.IsHidden(fpath) && acc.PathAccess(fpath, uid == aid) {
				if fi, _ := StatFile(fpath); fi != nil {
					vfiles = append(vfiles, fi)
					vpaths = append(vpaths, MakeFilePath(fpath))
				}
			}
		}
		return arg.Limit == 0 || len(ret.List) < arg.Limit
	})
	if ret.List, _, err = ScanFileInfoList(acc, session, vfiles, vpaths); err != nil {
		WriteError500(w, r, err, AECgpsrangelist)
		return
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func gpsscanAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		List []Puid_t `json:"list" yaml:"list" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []Store[GpsInfo] `json:"list" yaml:"list" xml:"list>tile"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECgpsscannoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECgpsscannodata)
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
			if val, ok := gpscache.Peek(puid); ok {
				var gst = Store[GpsInfo]{
					Puid: puid,
					Prop: val,
				}
				ret.List = append(ret.List, gst)
			} else {
				// check memory cache
				if exifcache.Has(puid) {
					continue // there are tags without GPS
				}
				// try to get from database
				var err error
				var est ExifStore
				if ok, err = session.ID(puid).Get(&est); err != nil {
					continue
				}
				if ok {
					exifcache.Poke(puid, est.Prop) // update cache
					if est.Prop.IsZero() {
						continue
					}
				} else {
					// try to extract from file
					func() (err error) {
						var file File
						if file, err = OpenFile(syspath); err != nil {
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
					// set to memory cache
					exifcache.Poke(puid, est.Prop)
					// prepare to set to database
					ests = append(ests, est)
				}
				if est.Prop.Latitude != 0 || est.Prop.Longitude != 0 {
					var gst Store[GpsInfo]
					gst.Puid = puid
					gst.Prop.FromProp(&est.Prop)
					ret.List = append(ret.List, gst)
					// set to GPS cache
					gpscache.Poke(puid, gst.Prop)
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

	WriteOK(w, r, &ret)
}

// The End.
