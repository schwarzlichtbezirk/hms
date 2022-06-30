package hms

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/dhowden/tag"
)

// TagEnum is descriptor for discs and tracks.
type TagEnum struct {
	Number int `json:"number,omitempty" yaml:"number,omitempty" xml:"number,omitempty"`
	Total  int `json:"total,omitempty" yaml:"total,omitempty" xml:"total,omitempty"`
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (te *TagEnum) IsZero() bool {
	return te.Number == 0 && te.Total == 0
}

// TagProp is Music file tags properties chunk.
type TagProp struct {
	Title    string  `json:"title,omitempty" yaml:"title,omitempty" xml:"title,omitempty"`
	Album    string  `json:"album,omitempty" yaml:"album,omitempty" xml:"album,omitempty"`
	Artist   string  `json:"artist,omitempty" yaml:"artist,omitempty" xml:"artist,omitempty"`
	Composer string  `json:"composer,omitempty" yaml:"composer,omitempty" xml:"composer,omitempty"`
	Genre    string  `json:"genre,omitempty" yaml:"genre,omitempty" xml:"genre,omitempty"`
	Year     int     `json:"year,omitempty" yaml:"year,omitempty" xml:"year,omitempty"`
	Track    TagEnum `json:"track,omitempty" yaml:"track,flow,omitempty" xml:"track,omitempty"`
	Disc     TagEnum `json:"disc,omitempty" yaml:"disc,flow,omitempty" xml:"disc,omitempty"`
	Lyrics   string  `json:"lyrics,omitempty" yaml:"lyrics,omitempty" xml:"lyrics,omitempty"`
	Comment  string  `json:"comment,omitempty" yaml:"comment,omitempty" xml:"comment,omitempty"`
}

// Setup fills fields from tags metadata.
func (tp *TagProp) Setup(m tag.Metadata) {
	tp.Title = m.Title()
	tp.Album = m.Album()
	tp.Artist = m.Artist()
	tp.Composer = m.Composer()
	tp.Genre = m.Genre()
	tp.Year = m.Year()
	tp.Track.Number, tp.Track.Total = m.Track()
	tp.Disc.Number, tp.Disc.Total = m.Disc()
	tp.Lyrics = m.Lyrics()
	tp.Comment = m.Comment()
}

// TagKit is music file tags properties kit.
type TagKit struct {
	FileProp `yaml:",inline"`
	PuidProp `yaml:",inline"`
	TmbProp  `yaml:",inline"`
	TagProp  `yaml:",inline"`
}

// Setup fills fields with given path.
// Puts into the cache nested at the tags thumbnail if it present.
func (tk *TagKit) Setup(syspath string, fi fs.FileInfo) {
	tk.FileProp.Setup(fi)
	tk.PuidProp.Setup(syspath)

	if file, err := OpenFile(syspath); err == nil {
		defer file.Close()
		if m, err := tag.ReadFrom(file); err == nil {
			tk.TagProp.Setup(m)
			if pic := m.Picture(); pic != nil {
				if cfg.FitEmbeddedTmb {
					var md *MediaData
					if md, err = GetCachedEmbThumb(bytes.NewReader(pic.Data), syspath); err == nil {
						tk.MTmbVal = md.Mime
						return
					}
				} else {
					tk.MTmbVal = GetMimeVal(pic.MIMEType)
					return
				}
			}
		}
	}
	tk.TmbProp = tmbdis
}

// GetTagTmb extracts embedded thumbnail from image file.
func GetTagTmb(syspath string) (md *MediaData, err error) {
	var file io.ReadSeekCloser
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err != nil {
		return
	}
	var pic *tag.Picture
	if pic = m.Picture(); pic == nil {
		err = ErrNotThumb
		return
	}
	if cfg.FitEmbeddedTmb {
		return GetCachedEmbThumb(bytes.NewReader(pic.Data), syspath)
	}
	md = &MediaData{
		Data: pic.Data,
		Mime: GetMimeVal(pic.MIMEType),
	}
	return
}

// MakeProp is file properties factory.
func MakeProp(syspath string, fi fs.FileInfo) Pather {
	if fi.IsDir() {
		var dk DirKit
		dk.Setup(syspath)
		return &dk
	}
	var ext = GetFileExt(syspath)
	if IsTypeID3(ext) {
		var tk TagKit
		tk.Setup(syspath, fi)
		return &tk
	} else if IsTypeEXIF(ext) {
		var ek ExifKit
		ek.Setup(syspath, fi)
		return &ek
	} else {
		var fk FileKit
		fk.Setup(syspath, fi)
		return &fk
	}
}

// The End.
