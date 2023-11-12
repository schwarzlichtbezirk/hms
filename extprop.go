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
	TagDis ExtTag = iota - 1 // file have no any tags

	TagNil  // tags have not scanned yet, indeterminate state
	TagImg  // image config only
	TagExif // image config + EXIF
	TagID3  // MP3/MP4/OGG/FLAC metadata
)

type ExtProp struct {
	Tags ExtTag `xorm:"tags" json:"tags" yaml:"tags" xml:"tags"`
	ETmb Mime_t `xorm:"etmb" json:"etmb" yaml:"etmb" xml:"etmb"` // embedded thumbnail

	Width   int           `xorm:"width" json:"width,omitempty" yaml:"width,omitempty" xml:"width,omitempty"`     // image width in pixels
	Height  int           `xorm:"height" json:"height,omitempty" yaml:"height,omitempty" xml:"height,omitempty"` // image height in pixels
	PBLen   time.Duration `xorm:"pblen" json:"pblen,omitempty" yaml:"pblen,omitempty" xml:"pblen,omitempty"`     // playback length
	BitRate int           `xorm:"bitrate" json:"bitrate,omitempty" yaml:"bitrate,omitempty" xml:"bitrate,omitempty"`
}

type ExtStat struct {
	ErrCount  uint64
	FileCount uint64
	ExtCount  uint64
	ImgCount  uint64
	ExifCount uint64
	Id3Count  uint64
	TmbCount  uint64
	Mp3Count  uint64
}

func TagsExtract(fpath string, session *Session, buf *StoreBuf, es *ExtStat, gettmb bool) (p any, xp ExtProp, err error) {
	defer func() {
		if err != nil {
			atomic.AddUint64(&es.ErrCount, 1)
		}
	}()
	atomic.AddUint64(&es.FileCount, 1)

	var puid, _ = PathStorePUID(session, fpath)
	var ext = GetFileExt(fpath)
	if IsTypeEXIF(ext) {
		var ek ExifKit
		var imc image.Config
		ek.Tags = TagDis
		ek.ETmb = MimeDis
		defer func() {
			p, xp = ek, ek.ExtProp
			buf.Push(session, ExtStore{
				Puid: puid,
				Prop: xp,
			})
			atomic.AddUint64(&es.ExtCount, 1)
		}()

		var file jnt.RFile
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		ek.Tags = TagImg // image config is exist
		ek.Width, ek.Height = imc.Width, imc.Height
		atomic.AddUint64(&es.ImgCount, 1)

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return
		}
		var x *exif.Exif
		if x, err = exif.Decode(file); err != nil {
			return
		}
		ek.Setup(x)
		if ek.ExifProp.IsZero() {
			return
		}
		ek.Tags = TagExif // EXIF is exist
		atomic.AddUint64(&es.ExifCount, 1)

		GpsCachePut(puid, ek.ExifProp)
		buf.Push(session, ExifStore{
			Puid: puid,
			Prop: ek.ExifProp,
		})

		if ek.ThumbJpegLen == 0 {
			return
		}
		ek.ETmb = MimeJpeg
		atomic.AddUint64(&es.TmbCount, 1)
		if gettmb {
			if pic, _ := x.JpegThumbnail(); pic != nil {
				var md = MediaData{
					Data: pic,
					Mime: MimeJpeg,
				}
				if fi, _ := file.Stat(); fi != nil {
					md.Time = fi.ModTime()
				}
				etmbcache.Poke(puid, md)
				ThumbCacheTrim()
			}
		}
	} else if IsTypeDecoded(ext) {
		var imc image.Config
		xp.Tags = TagDis
		xp.ETmb = MimeDis
		defer func() {
			p = xp
			buf.Push(session, ExtStore{
				Puid: puid,
				Prop: xp,
			})
			atomic.AddUint64(&es.ExtCount, 1)
		}()

		var file jnt.RFile
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		xp.Tags = TagImg // image config is exist
		xp.Width, xp.Height = imc.Width, imc.Height
		atomic.AddUint64(&es.ImgCount, 1)
	} else if IsTypeID3(ext) {
		var ik Id3Kit
		ik.Tags = TagDis
		ik.ETmb = MimeDis
		defer func() {
			p, xp = ik, ik.ExtProp
			buf.Push(session, ExtStore{
				Puid: puid,
				Prop: xp,
			})
			atomic.AddUint64(&es.ExtCount, 1)
		}()

		var file jnt.RFile
		if file, err = jnt.OpenFile(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		if IsTypeMp3(ext) {
			if ik.PBLen, ik.BitRate, err = Mp3Scan(file); err != nil {
				return
			}
			atomic.AddUint64(&es.Mp3Count, 1)
		}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return
		}
		var m tag.Metadata
		if m, err = tag.ReadFrom(file); err != nil {
			return
		}
		ik.Setup(m)
		if ik.Id3Prop.IsZero() {
			return
		}
		ik.Tags = TagID3 // ID3 is exist
		atomic.AddUint64(&es.Id3Count, 1)

		buf.Push(session, Id3Store{
			Puid: puid,
			Prop: ik.Id3Prop,
		})

		if ik.TmbMime == MimeDis {
			return
		}
		ik.ETmb = ik.TmbMime
		atomic.AddUint64(&es.TmbCount, 1)
		if gettmb {
			if pic := m.Picture(); pic != nil {
				var md = MediaData{
					Data: pic.Data,
					Mime: GetMimeVal(pic.MIMEType, pic.Ext),
				}
				if fi, _ := file.Stat(); fi != nil {
					md.Time = fi.ModTime()
				}
				etmbcache.Poke(puid, md)
				ThumbCacheTrim()
			}
		}
	}

	return
}

// The End.
