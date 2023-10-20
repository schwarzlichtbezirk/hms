package hms

import (
	"image"
	"io"
	"sync/atomic"
	"time"

	jnt "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/dhowden/tag"
	"github.com/rwcarlsen/goexif/exif"
)

type ExtTag int

const (
	TagDis  ExtTag = -1
	TagExif ExtTag = 1
	TagID3  ExtTag = 2
)

type ExtProp struct {
	Tags    ExtTag        `json:"tags" yaml:"tags" xml:"tags"`
	Thumb   Mime_t        `json:"thumb" yaml:"thumb" xml:"thumb"`
	Width   int           `json:"width,omitempty" yaml:"width,omitempty" xml:"width,omitempty"`
	Height  int           `json:"height,omitempty" yaml:"height,omitempty" xml:"height,omitempty"`
	Length  time.Duration `json:"length,omitempty" yaml:"length,omitempty" xml:"length,omitempty"`
	BitRate int           `xorm:"bitrate" json:"bitrate,omitempty" yaml:"bitrate,omitempty" xml:"bitrate,omitempty"`
}

type ExtStat struct {
	ErrCount  uint64
	FileCount uint64
	ExtCount  uint64
	ExifCount uint64
	Id3Count  uint64
	Mp3Count  uint64
}

func TagsExtract(fpath string, session *Session, buf *StoreBuf, es *ExtStat) (p any, err error) {
	defer func() {
		if err != nil {
			atomic.AddUint64(&es.ErrCount, 1)
		}
	}()
	atomic.AddUint64(&es.FileCount, 1)

	var puid, _ = PathCache.GetRev(fpath)
	var ext = GetFileExt(fpath)
	if IsTypeEXIF(ext) {
		var file jnt.File
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var ek ExifKit
		var imc image.Config

		if x, err := exif.Decode(file); err == nil {
			ek.Setup(x)
			if !ek.ExifProp.IsZero() {
				GpsCachePut(puid, ek.ExifProp)
				buf.Push(session, ExifStore{
					Puid: puid,
					Prop: ek.ExifProp,
				})
				ek.Tags = TagExif // EXIF is exist
				if ek.ThumbJpegLen > 0 {
					ek.Thumb = MimeJpeg
				}
				atomic.AddUint64(&es.ExifCount, 1)
			}
		}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return
		}
		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		ek.Width, ek.Height = imc.Width, imc.Height
		buf.Push(session, ExtStore{
			Puid: puid,
			Prop: ek.ExtProp,
		})
		atomic.AddUint64(&es.ExtCount, 1)
		p = ek
	} else if IsTypeDecoded(ext) {
		var file jnt.File
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var xp ExtProp
		var imc image.Config

		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		xp.Tags = 0
		xp.Width, xp.Height = imc.Width, imc.Height
		buf.Push(session, ExtStore{
			Puid: puid,
			Prop: xp,
		})
		atomic.AddUint64(&es.ExtCount, 1)
		p = xp
	} else if IsTypeID3(ext) {
		var file jnt.File
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var ik Id3Kit

		if m, err := tag.ReadFrom(file); err == nil {
			ik.Setup(m)
			if !ik.Id3Prop.IsZero() {
				buf.Push(session, Id3Store{
					Puid: puid,
					Prop: ik.Id3Prop,
				})
				ik.Tags = TagID3
				ik.Thumb = ik.TmbMime
				atomic.AddUint64(&es.Id3Count, 1)
			}
		}

		if IsTypeMp3(ext) {
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				return
			}
			if ik.Length, ik.BitRate, err = Mp3Scan(file); err != nil {
				return
			}
			atomic.AddUint64(&es.Mp3Count, 1)
		}

		buf.Push(session, ExtStore{
			Puid: puid,
			Prop: ik.ExtProp,
		})
		atomic.AddUint64(&es.ExtCount, 1)
		p = ik
	}

	return
}

// The End.
