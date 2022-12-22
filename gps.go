package hms

import (
	"encoding/xml"
	"io/fs"
	"math"
	"net/http"
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
func gpsrangeAPI(w http.ResponseWriter, r *http.Request, auth *Profile) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID   ID_t      `json:"aid" yaml:"aid" xml:"aid,attr"`
		Paths []MapPath `json:"paths" yaml:"paths" xml:"paths>path"`
		Limit int       `json:"limit,omitempty" yaml:"limit,omitempty" xml:"limit,omitempty"`
		Time1 Unix_t    `json:"time1,omitempty" yaml:"time1,omitempty" xml:"time1,omitempty"`
		Time2 Unix_t    `json:"time2,omitempty" yaml:"time2,omitempty" xml:"time2,omitempty"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []any `json:"list" yaml:"list" xml:"list>prop"`
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
		arg.Limit = cfg.RangeSearchLimit
	}

	var session = xormEngine.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECgpsrangenoacc)
		return
	}

	var vfiles []fs.FileInfo // verified file infos
	var vpaths []string      // verified paths
	gpscache.Range(func(puid Puid_t, gps GpsInfo) bool {
		var inc bool
		for _, mp := range arg.Paths {
			if !mp.Contains(gps.Latitude, gps.Longitude) {
				continue
			}
			if arg.Time1 > 0 && gps.DateTime < arg.Time1 {
				continue
			}
			if arg.Time2 > 0 && gps.DateTime > arg.Time2 {
				continue
			}
			inc = !mp.Eject
		}
		if inc {
			var fpath, _ = pathcache.GetDir(puid)
			if !prf.IsHidden(fpath) && prf.PathAccess(fpath, auth == prf) {
				if fi, _ := StatFile(fpath); fi != nil {
					vfiles = append(vfiles, fi)
					vpaths = append(vpaths, fpath)
				}
			}
		}
		return arg.Limit == 0 || len(ret.List) < arg.Limit
	})
	if ret.List, _, err = ScanFileInfoList(prf, session, vfiles, vpaths); err != nil {
		WriteError500(w, r, err, AECgpsrangelist)
		return
	}

	WriteOK(w, r, &ret)
}

// The End.
