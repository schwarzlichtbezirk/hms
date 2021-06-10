package hms

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/beevik/guid"
)

type Track struct {
	Time int64
	Name string
	Path string
}

type Playlist struct {
	Tracks []Track
	Dest   string // playlist file destination
	Title  string
}

type WPLMeta struct {
	Text    string `xml:",chardata"`
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type WPLMedia struct {
	Text string `xml:",chardata"`
	Src  string `xml:"src,attr"`
	Tid  string `xml:"tid,attr,omitempty"`
}

type WPL struct {
	XMLName xml.Name `xml:"smil"`
	Text    string   `xml:",chardata"`
	Head    struct {
		Text  string    `xml:",chardata"`
		Meta  []WPLMeta `xml:"meta"`
		Title string    `xml:"title"`
	} `xml:"head"`
	Body struct {
		Text string `xml:",chardata"`
		Seq  struct {
			Text  string     `xml:",chardata"`
			Media []WPLMedia `xml:"media"`
		} `xml:"seq"`
	} `xml:"body"`
}

var (
	ErrM3USign = errors.New("file does not starts with M3U signature")
)

func isURL(fpath string) bool {
	return strings.HasPrefix(fpath, "http://") || strings.HasPrefix(fpath, "https://")
}

func (pl *Playlist) ReadFrom(r io.Reader) (num int64, err error) {
	return pl.ReadM3U(r)
}

func (pl *Playlist) WriteTo(w io.Writer) (num int64, err error) {
	return pl.WriteM3U(w)
}

func (pl *Playlist) ReadM3U(r io.Reader) (num int64, err error) {
	var buf = bufio.NewReader(r)
	var line string

	var readline = func() {
		for {
			line, err = buf.ReadString('\n')
			num += int64(len(line))
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return
			}
			line = strings.TrimSpace(line)
			if len(line) > 0 {
				break
			}
		}
	}

	if readline(); err != nil {
		return
	}
	if line != "#EXTM3U" && line != utf8bom+"#EXTM3U" {
		return 0, ErrM3USign
	}

	for {
		var track Track

		if readline(); err != nil || len(line) == 0 {
			return
		}

		if strings.HasPrefix(line, "#EXTINF") {
			if _, err = fmt.Sscanf(line, "#EXTINF:%d,%s", &track.Time, &track.Name); err != nil {
				return
			}

			if readline(); err != nil || len(line) == 0 {
				return
			}
		}
		if filepath.IsAbs(line) || isURL(line) {
			track.Path = line
		} else if line[0] == filepath.Separator {
			track.Path = filepath.Join(filepath.VolumeName(pl.Dest), line)
		} else {
			track.Path = filepath.Join(pl.Dest, line)
		}
		pl.Tracks = append(pl.Tracks, track)
	}
}

func (pl *Playlist) WriteM3U(w io.Writer) (num int64, err error) {
	var n int
	if n, err = fmt.Fprintln(w, "#EXTM3U"); err != nil {
		num += int64(n)
		return
	}
	num += int64(n)
	const sep = string(filepath.Separator)
	var dir0 = filepath.Clean(pl.Dest)
	var dir1 = filepath.Dir(dir0)
	dir0 += sep
	dir1 += sep
	for _, track := range pl.Tracks {
		if track.Time > 0 {
			if n, err = fmt.Fprintf(w, "#EXTINF:%d,%s\n", track.Time, track.Name); err != nil {
				num += int64(n)
				return
			}
			num += int64(n)
		}
		var fpath = track.Path
		if strings.HasPrefix(fpath, dir0) || strings.HasPrefix(fpath, dir1) {
			if fpath, err = filepath.Rel(dir0, fpath); err != nil {
				return
			}
		}
		if n, err = fmt.Fprintln(w, fpath); err != nil {
			num += int64(n)
			return
		}
		num += int64(n)
	}
	return
}

func (pl *Playlist) WriteM3U8(w io.Writer) (num int64, err error) {
	var n int
	var n64 int64
	n, err = w.Write([]byte(utf8bom))
	num += int64(n)
	if err != nil {
		return
	}
	n64, err = pl.WriteM3U(w)
	num += n64
	return
}

func (pl *Playlist) ReadWPL(r io.Reader) (num int64, err error) {
	var body []byte
	if body, err = io.ReadAll(r); err != nil {
		return
	}
	num = int64(len(body))

	var wpl WPL
	if err = xml.Unmarshal(body, &wpl); err != nil {
		return
	}

	for _, m := range wpl.Body.Seq.Media {
		var track Track
		if filepath.IsAbs(m.Src) || isURL(m.Src) {
			track.Path = m.Src
		} else if m.Src[0] == filepath.Separator {
			track.Path = filepath.Join(filepath.VolumeName(pl.Dest), m.Src)
		} else {
			track.Path = filepath.Join(pl.Dest, m.Src)
		}
		pl.Tracks = append(pl.Tracks, track)
	}
	return
}

func (pl *Playlist) WriteWPL(w io.Writer) (num int64, err error) {
	var n int
	n, err = fmt.Fprintln(w, `<?wpl version="1.0"?>`)
	num += int64(n)
	if err != nil {
		return
	}

	var wpl WPL
	wpl.Head.Title = pl.Title
	wpl.Head.Meta = append(wpl.Head.Meta, WPLMeta{Name: "Generator", Content: "Home Media Server"})
	wpl.Head.Meta = append(wpl.Head.Meta, WPLMeta{Name: "ItemCount", Content: fmt.Sprintf("%d", len(pl.Tracks))})

	const sep = string(filepath.Separator)
	var dir0 = filepath.Clean(pl.Dest)
	var dir1 = filepath.Dir(dir0)
	dir0 += sep
	dir1 += sep
	for _, track := range pl.Tracks {
		var fpath = track.Path
		if strings.HasPrefix(fpath, dir0) || strings.HasPrefix(fpath, dir1) {
			if fpath, err = filepath.Rel(dir0, fpath); err != nil {
				return
			}
		}
		var guid = guid.New()
		wpl.Body.Seq.Media = append(wpl.Body.Seq.Media, WPLMedia{Src: fpath, Tid: "{" + guid.StringUpper() + "}"})
	}

	var body []byte
	if body, err = xml.MarshalIndent(&wpl, "", "    "); err != nil {
		return
	}
	n, err = w.Write(body)
	num = int64(n)
	return
}

// The End.
