package hms

import (
	"errors"
	"io"
	"time"

	"github.com/dhowden/tag"
	"github.com/tcolgate/mp3"
)

// Id3Prop is Music file tags properties chunk.
type Id3Prop struct {
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
	ThumbLen int    `xorm:"'thumblen'" json:"thumblen,omitempty" yaml:"thumblen,omitempty" xml:"thumblen,omitempty"`
	TmbMime  Mime_t `xorm:"'tmbmime'" json:"tmbmime,omitempty" yaml:"tmbmime,omitempty" xml:"tmbmime,omitempty"`
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (tp *Id3Prop) IsZero() bool {
	return tp.Title == "" && tp.Album == "" && tp.Artist == "" &&
		tp.Composer == "" && tp.Genre == "" && tp.Year == 0 &&
		tp.TrackNum == 0 && tp.TrackSum == 0 &&
		tp.DiscNum == 0 && tp.DiscSum == 0 &&
		tp.Lyrics == "" && tp.Comment == "" &&
		tp.ThumbLen == 0
}

// Setup fills fields from tags metadata.
func (tp *Id3Prop) Setup(m tag.Metadata) {
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
	if pic := m.Picture(); pic != nil {
		tp.ThumbLen = len(pic.Data)
		tp.TmbMime = GetMimeVal(pic.MIMEType, pic.Ext)
	} else {
		tp.ThumbLen = 0
		tp.TmbMime = MimeDis
	}
}

// Id3Extract trys to extract ID3 metadata from file.
func Id3Extract(session *Session, file io.ReadSeeker, puid Puid_t) (tp Id3Prop, err error) {
	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err != nil {
		return
	}

	tp.Setup(m)
	if tp.IsZero() {
		err = ErrEmptyID3
		return
	}
	Id3StoreSet(session, puid, tp) // update database
	return
}

// Id3ExtractThumb trys to extract thumbnail from file ID3 metadata.
func Id3ExtractThumb(syspath string) (md MediaData, err error) {
	// disable thumbnail if it not found
	defer func() {
		if md.Mime == MimeNil {
			md.Mime = MimeDis
		}
	}()

	var file RFile
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

// Mp3Scan scans file to calculate length of play and average bitrate.
// Bitrate is rounded up to kilobits.
func Mp3Scan(r io.Reader) (length time.Duration, bitrate int, err error) {
	var d = mp3.NewDecoder(r)
	var brm = map[mp3.FrameBitRate]int{}
	var f mp3.Frame
	var skipped int
	var ms float64

	for {
		if err = d.Decode(&f, &skipped); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				err = nil // end of file is reached in any case
			}
			break
		}
		var h = f.Header()
		if br := h.BitRate(); br != mp3.ErrInvalidBitrate {
			brm[br]++
		}
		if sr := f.Header().SampleRate(); sr != mp3.ErrInvalidSampleRate {
			ms += (1000 / float64(sr)) * float64(f.Samples())
		}
	}
	length = time.Duration(ms * float64(time.Millisecond))

	var n, ws float64
	for br, sn := range brm {
		n += float64(sn)
		ws += float64(br) * float64(sn)
	}
	if n > 0 {
		bitrate = int(ws/n+500) / 1000
	}
	return
}

// Id3Kit is music file tags properties kit.
type Id3Kit struct {
	ExtProp `xorm:"extends" yaml:",inline"`
	Id3Prop `xorm:"extends" yaml:",inline"`
}

// The End.
