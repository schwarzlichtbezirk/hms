package hms

import (
	"fmt"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

// EXIF tags properties chunk.
type ExifProp struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Photo
	Model        string  `json:"model,omitempty"`
	Make         string  `json:"make,omitempty"`
	Software     string  `json:"software,omitempty"`
	DateTime     int64   `json:"datetime,omitempty"`
	Orientation  int     `json:"orientation,omitempty"`
	ExposureTime string  `json:"exposuretime,omitempty"`
	ExposureProg int     `json:"exposureprog,omitempty"`
	FNumber      float64 `json:"fnumber,omitempty"`
	ISOSpeed     int     `json:"isospeed,omitempty"`
	ShutterSpeed float64 `json:"shutterspeed,omitempty"`
	Aperture     float64 `json:"aperture,omitempty"`
	ExposureBias float64 `json:"exposurebias,omitempty"`
	LightSource  int     `json:"lightsource,omitempty"`
	Focal        float64 `json:"focal,omitempty"`
	Focal35mm    int     `json:"focal35mm,omitempty"`
	DigitalZoom  float64 `json:"digitalzoom,omitempty"`
	Flash        int     `json:"flash,omitempty"`
	UniqueID     string  `json:"uniqueid,omitempty"`
	// GPS
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	Satelites string  `json:"satelites,omitempty"`
}

func ratfloat(t *tiff.Tag) float64 {
	var numer, denom, _ = t.Rat2(0)
	if denom != 0 {
		return float64(numer) / float64(denom)
	}
	return 0
}

// Fills fields from given EXIF structure.
func (ep *ExifProp) Setup(x *exif.Exif) {
	var err error
	var t *tiff.Tag

	if t, err = x.Get(exif.ImageWidth); err == nil {
		ep.Width, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ImageLength); err == nil {
		ep.Height, _ = t.Int(0)
	}
	if t, err = x.Get(exif.Model); err == nil {
		ep.Model, _ = t.StringVal()
	}
	if t, err = x.Get(exif.Make); err == nil {
		ep.Make, _ = t.StringVal()
	}
	if t, err = x.Get(exif.Software); err == nil {
		ep.Software, _ = t.StringVal()
	}
	if tm, err := x.DateTime(); err == nil {
		ep.DateTime = UnixJS(tm)
	}
	if t, err = x.Get(exif.Orientation); err == nil {
		ep.Orientation, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ExposureTime); err == nil {
		var numer, denom, _ = t.Rat2(0)
		ep.ExposureTime = fmt.Sprintf("%d/%d", numer, denom)
	}
	if t, err = x.Get(exif.ExposureProgram); err == nil {
		ep.ExposureProg, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FNumber); err == nil {
		ep.FNumber = ratfloat(t)
	}
	if t, err = x.Get(exif.ISOSpeedRatings); err == nil {
		ep.ISOSpeed, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ShutterSpeedValue); err == nil {
		ep.ShutterSpeed = ratfloat(t)
	}
	if t, err = x.Get(exif.ApertureValue); err == nil {
		ep.Aperture = ratfloat(t)
	}
	if t, err = x.Get(exif.ExposureBiasValue); err == nil {
		ep.ExposureBias = ratfloat(t)
	}
	if t, err = x.Get(exif.LightSource); err == nil {
		ep.LightSource, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FocalLength); err == nil {
		ep.Focal = ratfloat(t)
	}
	if t, err = x.Get(exif.FocalLengthIn35mmFilm); err == nil {
		ep.Focal35mm, _ = t.Int(0)
	}
	if t, err = x.Get(exif.DigitalZoomRatio); err == nil {
		ep.DigitalZoom = ratfloat(t)
	}
	if t, err = x.Get(exif.ImageLength); err == nil {
		ep.Height, _ = t.Int(0)
	}
	if t, err = x.Get(exif.Flash); err == nil {
		ep.Flash, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ImageUniqueID); err == nil {
		ep.UniqueID, _ = t.StringVal()
	}
	if lat, lng, err := x.LatLong(); err == nil {
		ep.Latitude, ep.Longitude = lat, lng
	}
	if t, err = x.Get(exif.GPSAltitude); err == nil {
		ep.Altitude = ratfloat(t)
		if t, err = x.Get(exif.GPSAltitudeRef); err == nil {
			var ref, _ = t.Int(0)
			if ref == 1 {
				ep.Altitude *= -1.0
			}
		}
	}
	if t, err = x.Get(exif.GPSSatelites); err == nil {
		ep.Satelites, _ = t.StringVal()
	}
}

// File with EXIF tags.
type ExifKit struct {
	StdProp
	TmbProp
	ExifProp
}

// Creates copy of it self.
func (ek *ExifKit) Clone() Proper {
	var c = *ek
	return &c
}

// Fills fields with given path.
func (ek *ExifKit) Setup(fpath string, fi os.FileInfo) {
	ek.StdProp.Setup(fi)

	if file, err := os.Open(fpath); err == nil {
		defer file.Close()
		if x, err := exif.Decode(file); err == nil {
			ek.ExifProp.Setup(x)
			if len(ek.Model) > 0 {
				ek.TypeVal = FT_photo
			}
			if pic, err := x.JpegThumbnail(); err == nil {
				ek.KTmbVal = ThumbName(fpath)
				thumbcache.Set(ek.KTmbVal, &ThumbElem{
					Data: pic,
					Mime: "image/jpeg",
				})
				ek.NTmbVal = TMB_cached
				return
			}
		}
	}
	ek.TmbProp.Setup(fpath)
}

func exifparsers() {
	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

// The End.
