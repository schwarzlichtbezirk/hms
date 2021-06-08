package hms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type Track struct {
	Time int64
	Name string
	Path string
}

type Playlist struct {
	Tracks []Track
	Dest   string // playlist file destination
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

// The End.
