package hms

import (
	"fmt"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

// ExifProp is EXIF tags properties chunk.
type ExifProp struct {
	Width  int `json:"width,omitempty" yaml:"width,omitempty"`
	Height int `json:"height,omitempty" yaml:"height,omitempty"`
	// Photo
	Model        string  `json:"model,omitempty" yaml:"model,omitempty"`
	Make         string  `json:"make,omitempty" yaml:"make,omitempty"`
	Software     string  `json:"software,omitempty" yaml:"software,omitempty"`
	DateTime     int64   `json:"datetime,omitempty" yaml:"datetime,omitempty"`
	Orientation  int     `json:"orientation,omitempty" yaml:"orientation,omitempty"`
	ExposureTime string  `json:"exposuretime,omitempty" yaml:"exposuretime,omitempty"`
	ExposureProg int     `json:"exposureprog,omitempty" yaml:"exposureprog,omitempty"`
	FNumber      float64 `json:"fnumber,omitempty" yaml:"fnumber,omitempty"`
	ISOSpeed     int     `json:"isospeed,omitempty" yaml:"isospeed,omitempty"`
	ShutterSpeed float64 `json:"shutterspeed,omitempty" yaml:"shutterspeed,omitempty"`
	Aperture     float64 `json:"aperture,omitempty" yaml:"aperture,omitempty"`
	ExposureBias float64 `json:"exposurebias,omitempty" yaml:"exposurebias,omitempty"`
	LightSource  int     `json:"lightsource,omitempty" yaml:"lightsource,omitempty"`
	Focal        float64 `json:"focal,omitempty" yaml:"focal,omitempty"`
	Focal35mm    int     `json:"focal35mm,omitempty" yaml:"focal35mm,omitempty"`
	DigitalZoom  float64 `json:"digitalzoom,omitempty" yaml:"digitalzoom,omitempty"`
	Flash        int     `json:"flash,omitempty" yaml:"flash,omitempty"`
	UniqueID     string  `json:"uniqueid,omitempty" yaml:"uniqueid,omitempty"`
	ThumbJpegLen int     `json:"thumbjpeglen,omitempty" yaml:"thumbjpeglen,omitempty"`
	// GPS
	Latitude  float64 `json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty" yaml:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty" yaml:"altitude,omitempty"`
	Satelites string  `json:"satelites,omitempty" yaml:"satelites,omitempty"`
}

func ratfloat(t *tiff.Tag) float64 {
	var numer, denom, _ = t.Rat2(0)
	if denom != 0 {
		return float64(numer) / float64(denom)
	}
	return 0
}

// Setup fills fields from given EXIF structure.
func (ep *ExifProp) Setup(x *exif.Exif) {
	var err error
	var t *tiff.Tag

	if t, err = x.Get(exif.PixelXDimension); err == nil {
		ep.Width, _ = t.Int(0)
	}
	if t, err = x.Get(exif.PixelYDimension); err == nil {
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
	if t, err = x.Get(exif.ThumbJPEGInterchangeFormatLength); err == nil {
		ep.ThumbJpegLen, _ = t.Int(0)
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

// ExifKit is file with EXIF tags.
type ExifKit struct {
	FileProp
	TmbProp
	ExifProp
}

// Setup fills fields with given path.
func (ek *ExifKit) Setup(syspath string, fi os.FileInfo) {
	ek.FileProp.Setup(fi)

	if file, err := OpenFile(syspath); err == nil {
		defer file.Close()
		if x, err := exif.Decode(file); err == nil {
			ek.ExifProp.Setup(x)
			if cfg.UseEmbeddedTmb {
				if pic, err := x.JpegThumbnail(); err == nil {
					ek.PUIDVal = syspathcache.Cache(syspath)
					ek.SetTmb(TMBcached, "image/jpeg")
					thumbcache.Set(ek.PUIDVal, &MediaData{
						Data: pic,
						Mime: ek.MTmbVal,
					})
					return
				}
			}
		}
	}
	ek.TmbProp.Setup(syspath)
}

// GetExifTmb extracts JPEG thumbnail from the image file.
func GetExifTmb(syspath string) (md *MediaData, err error) {
	var file VFile
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var x *exif.Exif
	if x, err = exif.Decode(file); err == nil {
		var pic []byte
		if pic, err = x.JpegThumbnail(); err == nil {
			md = &MediaData{
				Data: pic,
				Mime: "image/jpeg",
			}
			return
		}
	}
	return
}

func exifparsers() {
	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

// The End.
