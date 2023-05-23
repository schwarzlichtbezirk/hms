package hms

import (
	"io"
	"io/fs"

	"github.com/dhowden/tag"
)

// TagProp is Music file tags properties chunk.
type TagProp struct {
	Title    string `json:"title,omitempty" yaml:"title,omitempty" xml:"title,omitempty"`
	Album    string `json:"album,omitempty" yaml:"album,omitempty" xml:"album,omitempty"`
	Artist   string `json:"artist,omitempty" yaml:"artist,omitempty" xml:"artist,omitempty"`
	Composer string `json:"composer,omitempty" yaml:"composer,omitempty" xml:"composer,omitempty"`
	Genre    string `json:"genre,omitempty" yaml:"genre,omitempty" xml:"genre,omitempty"`
	Year     int    `json:"year,omitempty" yaml:"year,omitempty" xml:"year,omitempty"`
	TrackNum int    `xorm:"'tracknum'" json:"tracknum,omitempty" yaml:"tracknum,flow,omitempty" xml:"tracknum,omitempty,attr"`
	TrackSum int    `xorm:"'tracksum'" json:"tracksum,omitempty" yaml:"tracksum,flow,omitempty" xml:"tracksum,omitempty,attr"`
	DiscNum  int    `xorm:"'discnum'" json:"discnum,omitempty" yaml:"discnum,flow,omitempty" xml:"discnum,omitempty,attr"`
	DiscSum  int    `xorm:"'discsum'" json:"discsum,omitempty" yaml:"discsum,flow,omitempty" xml:"discsum,omitempty,attr"`
	Lyrics   string `json:"lyrics,omitempty" yaml:"lyrics,omitempty" xml:"lyrics,omitempty"`
	Comment  string `json:"comment,omitempty" yaml:"comment,omitempty" xml:"comment,omitempty"`
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (tp *TagProp) IsZero() bool {
	return tp.Title == "" && tp.Album == "" && tp.Artist == "" &&
		tp.Composer == "" && tp.Genre == "" && tp.Year == 0 &&
		tp.TrackNum == 0 && tp.TrackSum == 0 &&
		tp.DiscNum == 0 && tp.DiscSum == 0 &&
		tp.Lyrics == "" && tp.Comment == ""
}

// Setup fills fields from tags metadata.
func (tp *TagProp) Setup(m tag.Metadata) {
	tp.Title = m.Title()
	tp.Album = m.Album()
	tp.Artist = m.Artist()
	tp.Composer = m.Composer()
	tp.Genre = m.Genre()
	tp.Year = m.Year()
	tp.TrackNum, tp.TrackSum = m.Track()
	tp.DiscNum, tp.DiscSum = m.Disc()
	tp.Lyrics = m.Lyrics()
	tp.Comment = m.Comment()
}

func (tp *TagProp) Extract(syspath string) (err error) {
	var r io.ReadSeekCloser
	if r, err = OpenFile(syspath); err != nil {
		return
	}
	defer r.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(r); err != nil {
		return
	}

	tp.Setup(m)
	return
}

func ExtractThumbID3(syspath string) (md MediaData, err error) {
	// disable thumbnail if it not found
	defer func() {
		if md.Mime == MimeNil {
			md.Mime = MimeDis
		}
	}()

	var file File
	if file, err = OpenFile(syspath); err != nil {
		return
	}
	defer file.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err != nil {
		return
	}

	var pic *tag.Picture
	if pic = m.Picture(); pic == nil {
		err = ErrNoThumb
		return
	}

	md.Data = pic.Data
	md.Mime = GetMimeVal(pic.MIMEType, pic.Ext)
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}
	return
}

// TagKit is music file tags properties kit.
type TagKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	TileProp `xorm:"extends" yaml:",inline"`
	TagProp  `xorm:"extends" yaml:",inline"`
}

// Setup fills fields with given path.
// Puts into the cache nested at the tags thumbnail if it present.
func (tk *TagKit) Setup(session *Session, syspath string, fi fs.FileInfo) {
	tk.PuidProp.Setup(session, syspath)
	tk.FileProp.Setup(fi)
	if tp, ok := tilecache.Peek(tk.PUID); ok {
		tk.TileProp = *tp
	}
	tk.TagProp, _ = TagStoreGet(session, tk.PUID)
}

// The End.
