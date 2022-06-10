package hms

import (
	"fmt"
	"io/fs"

	"github.com/disintegration/gift"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

// EXIF image orientation constants.
const (
	// orientation: normal
	OrientNormal = 1
	// orientation: horizontal reversed
	OrientHorzReversed = 2
	// orientation: flipped
	OrientFlipped = 3
	// orientation: flipped & horizontal reversed
	OrientFlipHorzReversed = 4
	// orientation: clockwise turned & horizontal reversed
	OrientCwHorzReversed = 5
	// orientation: clockwise turned
	OrientCw = 6
	// orientation: anticlockwise turned & horizontal reversed
	OrientAcwHorzReversed = 7
	// orientation: anticlockwise turned
	OrientAcw = 8
)

// AddOrientFilter appends filters to bring image to normal orientation.
func AddOrientFilter(flt []gift.Filter, orientation int) []gift.Filter {
	switch orientation {
	case OrientHorzReversed: // orientation: horizontal reversed
		flt = append(flt, gift.FlipHorizontal())
	case OrientFlipped: // orientation: flipped
		flt = append(flt, gift.Rotate180())
	case OrientFlipHorzReversed: // orientation: flipped & horizontal reversed
		flt = append(flt, gift.Rotate180())
		flt = append(flt, gift.FlipHorizontal())
	case OrientCwHorzReversed: // orientation: clockwise turned & horizontal reversed
		flt = append(flt, gift.Rotate270())
		flt = append(flt, gift.FlipHorizontal())
	case OrientCw: // clockwise turned
		flt = append(flt, gift.Rotate270())
	case OrientAcwHorzReversed: // orientation: anticlockwise turned & horizontal reversed
		flt = append(flt, gift.Rotate90())
		flt = append(flt, gift.FlipHorizontal())
	case OrientAcw: // anticlockwise turned
		flt = append(flt, gift.Rotate90())
	}
	return flt
}

// ExifProp is EXIF tags properties chunk.
type ExifProp struct {
	Width  int `json:"width,omitempty" yaml:"width,omitempty" xml:"width,omitempty"`
	Height int `json:"height,omitempty" yaml:"height,omitempty" xml:"height,omitempty"`
	// Photo
	Model        string  `json:"model,omitempty" yaml:"model,omitempty" xml:"model,omitempty"`
	Make         string  `json:"make,omitempty" yaml:"make,omitempty" xml:"make,omitempty"`
	Software     string  `json:"software,omitempty" yaml:"software,omitempty" xml:"software,omitempty"`
	DateTime     unix_t  `json:"datetime,omitempty" yaml:"datetime,omitempty" xml:"datetime,omitempty"`
	Orientation  int     `json:"orientation,omitempty" yaml:"orientation,omitempty" xml:"orientation,omitempty"`
	ExposureTime string  `json:"exposuretime,omitempty" yaml:"exposuretime,omitempty" xml:"exposuretime,omitempty"`
	ExposureProg int     `json:"exposureprog,omitempty" yaml:"exposureprog,omitempty" xml:"exposureprog,omitempty"`
	FNumber      float32 `json:"fnumber,omitempty" yaml:"fnumber,omitempty" xml:"fnumber,omitempty"`
	ISOSpeed     int     `json:"isospeed,omitempty" yaml:"isospeed,omitempty" xml:"isospeed,omitempty"`
	ShutterSpeed float32 `json:"shutterspeed,omitempty" yaml:"shutterspeed,omitempty" xml:"shutterspeed,omitempty"`
	Aperture     float32 `json:"aperture,omitempty" yaml:"aperture,omitempty" xml:"aperture,omitempty"`
	ExposureBias float32 `json:"exposurebias,omitempty" yaml:"exposurebias,omitempty" xml:"exposurebias,omitempty"`
	LightSource  int     `json:"lightsource,omitempty" yaml:"lightsource,omitempty" xml:"lightsource,omitempty"`
	Focal        float32 `json:"focal,omitempty" yaml:"focal,omitempty" xml:"focal,omitempty"`
	Focal35mm    int     `json:"focal35mm,omitempty" yaml:"focal35mm,omitempty" xml:"focal35mm,omitempty"`
	DigitalZoom  float32 `json:"digitalzoom,omitempty" yaml:"digitalzoom,omitempty" xml:"digitalzoom,omitempty"`
	Flash        int     `json:"flash,omitempty" yaml:"flash,omitempty" xml:"flash,omitempty"`
	UniqueID     string  `json:"uniqueid,omitempty" yaml:"uniqueid,omitempty" xml:"uniqueid,omitempty"`
	ThumbJpegLen int     `json:"thumbjpeglen,omitempty" yaml:"thumbjpeglen,omitempty" xml:"thumbjpeglen,omitempty"`
	// GPS
	Latitude   float64 `json:"latitude,omitempty" yaml:"latitude,omitempty" xml:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty" yaml:"longitude,omitempty" xml:"longitude,omitempty"`
	Altitude   float32 `json:"altitude,omitempty" yaml:"altitude,omitempty" xml:"altitude,omitempty"`
	Satellites string  `json:"satellites,omitempty" yaml:"satellites,omitempty" xml:"satellites,omitempty"`
	// private
	thumb MediaData
}

func ratfloat32(t *tiff.Tag) float32 {
	if numer, denom, _ := t.Rat2(0); denom != 0 {
		return float32(numer) / float32(denom)
	}
	return 0
}

func ratfloat64(t *tiff.Tag) float64 {
	if numer, denom, _ := t.Rat2(0); denom != 0 {
		return float64(numer) / float64(denom)
	}
	return 0
}

// Setup fills fields from given EXIF structure.
func (ep *ExifProp) Setup(x *exif.Exif) {
	var err error
	var t *tiff.Tag
	var pic []byte

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
		ep.FNumber = ratfloat32(t)
	}
	if t, err = x.Get(exif.ISOSpeedRatings); err == nil {
		ep.ISOSpeed, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ShutterSpeedValue); err == nil {
		ep.ShutterSpeed = ratfloat32(t)
	}
	if t, err = x.Get(exif.ApertureValue); err == nil {
		ep.Aperture = ratfloat32(t)
	}
	if t, err = x.Get(exif.ExposureBiasValue); err == nil {
		ep.ExposureBias = ratfloat32(t)
	}
	if t, err = x.Get(exif.LightSource); err == nil {
		ep.LightSource, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FocalLength); err == nil {
		ep.Focal = ratfloat32(t)
	}
	if t, err = x.Get(exif.FocalLengthIn35mmFilm); err == nil {
		ep.Focal35mm, _ = t.Int(0)
	}
	if t, err = x.Get(exif.DigitalZoomRatio); err == nil {
		ep.DigitalZoom = ratfloat32(t)
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
	if pic, err = x.JpegThumbnail(); err == nil {
		ep.thumb.Data = pic
		ep.thumb.Mime = MimeJpeg
	}
	if lat, lon, err := x.LatLong(); err == nil {
		ep.Latitude, ep.Longitude = lat, lon
	}
	if t, err = x.Get(exif.GPSAltitude); err == nil {
		ep.Altitude = ratfloat32(t)
		if t, err = x.Get(exif.GPSAltitudeRef); err == nil {
			var ref, _ = t.Int(0)
			if ref == 1 {
				ep.Altitude *= -1.0
			}
		}
	}
	if t, err = x.Get(exif.GPSSatelites); err == nil {
		ep.Satellites, _ = t.StringVal()
	}
}

// ExifKit is file with EXIF tags.
type ExifKit struct {
	FileProp `yaml:",inline"`
	TmbProp  `yaml:",inline"`
	ExifProp `yaml:",inline"`
}

// Setup fills fields with given path.
func (ek *ExifKit) Setup(syspath string, fi fs.FileInfo) {
	ek.FileProp.Setup(fi)

	if file, err := OpenFile(syspath); err == nil {
		defer file.Close()
		if x, err := exif.Decode(file); err == nil {
			ek.ExifProp.Setup(x)
			if ek.Latitude != 0 && ek.Longitude != 0 {
				defer func() {
					gpscache.Store(ek.PUIDVal, &GpsInfo{
						DateTime:  ek.DateTime,
						Latitude:  ek.Latitude,
						Longitude: ek.Longitude,
						Altitude:  ek.Altitude,
					})
				}()
			}
			if cfg.UseEmbeddedTmb && ek.ThumbJpegLen > 0 {
				ek.PUIDVal = syspathcache.Cache(syspath)
				ek.SetTmb(MimeJpeg)
				return
			}
		}
	}
	ek.TmbProp.Setup(syspath)
}

func exifparsers() {
	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

// The End.
