package hms

import (
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

// ExifProp is EXIF tags properties chunk.
type ExifProp struct {
	Width  int `json:"width,omitempty" yaml:"width,omitempty" xml:"width,omitempty"`
	Height int `json:"height,omitempty" yaml:"height,omitempty" xml:"height,omitempty"`
	// Photo
	Model        string    `xorm:"'model'" json:"model,omitempty" yaml:"model,omitempty" xml:"model,omitempty"`
	Make         string    `xorm:"'make'" json:"make,omitempty" yaml:"make,omitempty" xml:"make,omitempty"`
	Software     string    `xorm:"'software'" json:"software,omitempty" yaml:"software,omitempty" xml:"software,omitempty"`
	DateTime     time.Time `xorm:"'datetime'" json:"datetime,omitempty" yaml:"datetime,omitempty" xml:"datetime,omitempty"`
	Orientation  int       `xorm:"'orientation'" json:"orientation,omitempty" yaml:"orientation,omitempty" xml:"orientation,omitempty"`
	ExposureTime string    `xorm:"'exposure_time'" json:"exposuretime,omitempty" yaml:"exposuretime,omitempty" xml:"exposuretime,omitempty"`
	ExposureProg int       `xorm:"'exposure_prog'" json:"exposureprog,omitempty" yaml:"exposureprog,omitempty" xml:"exposureprog,omitempty"`
	FNumber      float32   `xorm:"'fnumber'" json:"fnumber,omitempty" yaml:"fnumber,omitempty" xml:"fnumber,omitempty"`
	ISOSpeed     int       `xorm:"'iso_speed'" json:"isospeed,omitempty" yaml:"isospeed,omitempty" xml:"isospeed,omitempty"`
	ShutterSpeed float32   `xorm:"'shutter_speed'" json:"shutterspeed,omitempty" yaml:"shutterspeed,omitempty" xml:"shutterspeed,omitempty"`
	Aperture     float32   `xorm:"'aperture'" json:"aperture,omitempty" yaml:"aperture,omitempty" xml:"aperture,omitempty"`
	ExposureBias float32   `xorm:"'exposure_bias'" json:"exposurebias,omitempty" yaml:"exposurebias,omitempty" xml:"exposurebias,omitempty"`
	LightSource  int       `xorm:"'light_source'" json:"lightsource,omitempty" yaml:"lightsource,omitempty" xml:"lightsource,omitempty"`
	Focal        float32   `xorm:"'focal'" json:"focal,omitempty" yaml:"focal,omitempty" xml:"focal,omitempty"`
	Focal35mm    int       `xorm:"'focal35mm'" json:"focal35mm,omitempty" yaml:"focal35mm,omitempty" xml:"focal35mm,omitempty"`
	DigitalZoom  float32   `xorm:"'digital_zoom'" json:"digitalzoom,omitempty" yaml:"digitalzoom,omitempty" xml:"digitalzoom,omitempty"`
	Flash        int       `xorm:"'flash'" json:"flash,omitempty" yaml:"flash,omitempty" xml:"flash,omitempty"`
	UniqueID     string    `xorm:"'unique_id'" json:"uniqueid,omitempty" yaml:"uniqueid,omitempty" xml:"uniqueid,omitempty"`
	ThumbJpegLen int       `xorm:"'thumb_jpeg_len'" json:"thumbjpeglen,omitempty" yaml:"thumbjpeglen,omitempty" xml:"thumbjpeglen,omitempty"`
	// GPS
	Latitude   float64 `xorm:"'latitude'" json:"latitude,omitempty" yaml:"latitude,omitempty" xml:"latitude,omitempty"`
	Longitude  float64 `xorm:"'longitude'" json:"longitude,omitempty" yaml:"longitude,omitempty" xml:"longitude,omitempty"`
	Altitude   float32 `xorm:"'altitude'" json:"altitude,omitempty" yaml:"altitude,omitempty" xml:"altitude,omitempty"`
	Satellites string  `xorm:"'satellites'" json:"satellites,omitempty" yaml:"satellites,omitempty" xml:"satellites,omitempty"`
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (ep *ExifProp) IsZero() bool {
	return ep.Width == 0 && ep.Height == 0 && ep.Model == "" &&
		ep.Make == "" && ep.Software == "" && ep.DateTime.IsZero() &&
		ep.Orientation == 0 && ep.ExposureTime == "" && ep.ExposureProg == 0 &&
		ep.FNumber == 0 && ep.ISOSpeed == 0 && ep.ShutterSpeed == 0 &&
		ep.Aperture == 0 && ep.ExposureBias == 0 && ep.LightSource == 0 &&
		ep.Focal == 0 && ep.Focal35mm == 0 && ep.DigitalZoom == 0 &&
		ep.Flash == 0 && ep.UniqueID == "" && ep.ThumbJpegLen == 0 &&
		ep.Latitude == 0 && ep.Longitude == 0 && ep.Altitude == 0 &&
		ep.Satellites == ""
}

func RatFloat32(t *tiff.Tag) float32 {
	if numer, denom, _ := t.Rat2(0); denom != 0 {
		return float32(numer) / float32(denom)
	}
	return 0
}

func RatFloat64(t *tiff.Tag) float64 {
	if numer, denom, _ := t.Rat2(0); denom != 0 {
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
		ep.DateTime = tm
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
		ep.FNumber = RatFloat32(t)
	}
	if t, err = x.Get(exif.ISOSpeedRatings); err == nil {
		ep.ISOSpeed, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ShutterSpeedValue); err == nil {
		ep.ShutterSpeed = RatFloat32(t)
	}
	if t, err = x.Get(exif.ApertureValue); err == nil {
		ep.Aperture = RatFloat32(t)
	}
	if t, err = x.Get(exif.ExposureBiasValue); err == nil {
		ep.ExposureBias = RatFloat32(t)
	}
	if t, err = x.Get(exif.LightSource); err == nil {
		ep.LightSource, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FocalLength); err == nil {
		ep.Focal = RatFloat32(t)
	}
	if t, err = x.Get(exif.FocalLengthIn35mmFilm); err == nil {
		ep.Focal35mm, _ = t.Int(0)
	}
	if t, err = x.Get(exif.DigitalZoomRatio); err == nil {
		ep.DigitalZoom = RatFloat32(t)
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
	if lat, lon, err := x.LatLong(); err == nil {
		ep.Latitude, ep.Longitude = lat, lon
	}
	if t, err = x.Get(exif.GPSAltitude); err == nil {
		ep.Altitude = RatFloat32(t)
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

func (ep *ExifProp) Extract(syspath string) (err error) {
	var r io.ReadSeekCloser
	if r, err = OpenFile(syspath); err != nil {
		return
	}
	defer r.Close()

	var x *exif.Exif
	if x, err = exif.Decode(r); err != nil {
		return
	}

	ep.Setup(x)
	return
}

func ExtractThumbEXIF(syspath string) (md MediaData, err error) {
	// disable thumbnail if it not found
	defer func() {
		if md.Mime == MimeNil {
			md.Mime = MimeDis
		}
	}()

	var r io.ReadSeekCloser
	if r, err = OpenFile(syspath); err != nil {
		return
	}
	defer r.Close()

	var x *exif.Exif
	if x, err = exif.Decode(r); err != nil {
		return
	}

	var pic []byte
	if pic, err = x.JpegThumbnail(); err != nil {
		err = ErrNoThumb // set err to 'no thumbnail'
		return
	}

	md.Data = pic
	md.Mime = MimeJpeg
	return
}

// ExifKit is file with EXIF tags.
type ExifKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	TileProp `xorm:"extends" yaml:",inline"`
	ExifProp `xorm:"extends" yaml:",inline"`
}

// Setup fills fields with given path.
func (ek *ExifKit) Setup(session *Session, syspath string, fi fs.FileInfo) {
	ek.PuidProp.Setup(session, syspath)
	ek.FileProp.Setup(fi)
	if tp, ok := tilecache.Peek(ek.PUID); ok {
		ek.TileProp = *tp
	}
	ek.ExifProp, _ = ExifStoreGet(session, ek.PUID)
}

func init() {
	// Optionally register camera makenote data parsing - currently
	// Nikon and Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

// The End.
