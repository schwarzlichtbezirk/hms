package hms

import (
	"fmt"
	"io"
	"time"

	jnt "github.com/schwarzlichtbezirk/joint"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

// ExifProp is EXIF tags properties chunk.
type ExifProp struct {
	ImgWdh int `json:"imgwdh,omitempty" yaml:"imgwdh,omitempty" xml:"imgwdh,omitempty"`
	ImgHgt int `json:"imghgt,omitempty" yaml:"imghgt,omitempty" xml:"imghgt,omitempty"`
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
func (tp *ExifProp) IsZero() bool {
	return tp.ImgWdh == 0 && tp.ImgHgt == 0 && tp.Model == "" &&
		tp.Make == "" && tp.Software == "" && tp.DateTime.IsZero() &&
		tp.Orientation == 0 && tp.ExposureTime == "" && tp.ExposureProg == 0 &&
		tp.FNumber == 0 && tp.ISOSpeed == 0 && tp.ShutterSpeed == 0 &&
		tp.Aperture == 0 && tp.ExposureBias == 0 && tp.LightSource == 0 &&
		tp.Focal == 0 && tp.Focal35mm == 0 && tp.DigitalZoom == 0 &&
		tp.Flash == 0 && tp.UniqueID == "" && tp.ThumbJpegLen == 0 &&
		tp.Latitude == 0 && tp.Longitude == 0 && tp.Altitude == 0 &&
		tp.Satellites == ""
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
func (tp *ExifProp) Setup(x *exif.Exif) {
	var err error
	var t *tiff.Tag

	if t, err = x.Get(exif.ImageWidth); err == nil {
		tp.ImgWdh, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ImageLength); err == nil {
		tp.ImgHgt, _ = t.Int(0)
	}
	if t, err = x.Get(exif.Model); err == nil {
		tp.Model, _ = t.StringVal()
	}
	if t, err = x.Get(exif.Make); err == nil {
		tp.Make, _ = t.StringVal()
	}
	if t, err = x.Get(exif.Software); err == nil {
		tp.Software, _ = t.StringVal()
	}
	if tm, err := x.DateTime(); err == nil {
		tp.DateTime = tm
	}
	if t, err = x.Get(exif.Orientation); err == nil {
		tp.Orientation, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ExposureTime); err == nil {
		var numer, denom, _ = t.Rat2(0)
		tp.ExposureTime = fmt.Sprintf("%d/%d", numer, denom)
	}
	if t, err = x.Get(exif.ExposureProgram); err == nil {
		tp.ExposureProg, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FNumber); err == nil {
		tp.FNumber = RatFloat32(t)
	}
	if t, err = x.Get(exif.ISOSpeedRatings); err == nil {
		tp.ISOSpeed, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ShutterSpeedValue); err == nil {
		tp.ShutterSpeed = RatFloat32(t)
	}
	if t, err = x.Get(exif.ApertureValue); err == nil {
		tp.Aperture = RatFloat32(t)
	}
	if t, err = x.Get(exif.ExposureBiasValue); err == nil {
		tp.ExposureBias = RatFloat32(t)
	}
	if t, err = x.Get(exif.LightSource); err == nil {
		tp.LightSource, _ = t.Int(0)
	}
	if t, err = x.Get(exif.FocalLength); err == nil {
		tp.Focal = RatFloat32(t)
	}
	if t, err = x.Get(exif.FocalLengthIn35mmFilm); err == nil {
		tp.Focal35mm, _ = t.Int(0)
	}
	if t, err = x.Get(exif.DigitalZoomRatio); err == nil {
		tp.DigitalZoom = RatFloat32(t)
	}
	if t, err = x.Get(exif.Flash); err == nil {
		tp.Flash, _ = t.Int(0)
	}
	if t, err = x.Get(exif.ImageUniqueID); err == nil {
		tp.UniqueID, _ = t.StringVal()
	}
	if t, err = x.Get(exif.ThumbJPEGInterchangeFormatLength); err == nil {
		tp.ThumbJpegLen, _ = t.Int(0)
	}
	if lat, lon, err := x.LatLong(); err == nil {
		tp.Latitude, tp.Longitude = lat, lon
	}
	if t, err = x.Get(exif.GPSAltitude); err == nil {
		tp.Altitude = RatFloat32(t)
		if t, err = x.Get(exif.GPSAltitudeRef); err == nil {
			var ref, _ = t.Int(0)
			if ref == 1 {
				tp.Altitude *= -1.0
			}
		}
	}
	if t, err = x.Get(exif.GPSSatelites); err == nil {
		tp.Satellites, _ = t.StringVal()
	}
}

// ExifExtract trys to extract EXIF metadata from file.
func ExifExtract(session *Session, file io.Reader, puid Puid_t) (tp ExifProp, err error) {
	var x *exif.Exif
	if x, err = exif.Decode(file); err != nil {
		return
	}

	tp.Setup(x)
	if tp.IsZero() {
		err = ErrEmptyExif
		return
	}
	ExifStoreSet(session, puid, tp) // update database
	return
}

func ExtractThumbEXIF(syspath string) (md MediaData, err error) {
	// disable thumbnail if it not found
	defer func() {
		if md.Mime == MimeNil {
			md.Mime = MimeDis
		}
	}()

	var file jnt.RFile
	if file, err = jnt.OpenFile(syspath); err != nil {
		return
	}
	defer file.Close()

	var x *exif.Exif
	if x, err = exif.Decode(file); err != nil {
		return
	}

	var pic []byte
	if pic, err = x.JpegThumbnail(); err != nil {
		err = ErrNoThumb // set err to 'no thumbnail'
		return
	}

	md.Data = pic
	md.Mime = MimeJpeg
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}
	return
}

// ExifKit is file with EXIF tags.
type ExifKit struct {
	ExtProp  `xorm:"extends" yaml:",inline"`
	ExifProp `xorm:"extends" yaml:",inline"`
}

func init() {
	// Optionally register camera makenote data parsing - currently
	// Nikon and Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

// The End.
