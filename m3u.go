package hms

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
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
	dest   string // playlist file destination
}

var (
	ErrM3USign = errors.New("file does not starts with M3U signature")
)

func isURL(fpath string) bool {
	return strings.HasPrefix(fpath, "http://") || strings.HasPrefix(fpath, "https://")
}

func (pl *Playlist) ReadFrom(r io.Reader) (n int64, err error) {
	var buf = bufio.NewReader(r)
	var line string

	if line, err = buf.ReadString('\n'); err != nil {
		return
	}
	if line != "#EXTM3U" {
		return 0, ErrM3USign
	}
	for {
		var track Track
		if line, err = buf.ReadString('\n'); err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		if strings.HasPrefix(line, "#EXTINF") {
			if _, err = fmt.Sscanf(line, "#EXTINF:%d,%s", &track.Time, &track.Name); err != nil {
				return
			}
			if line, err = buf.ReadString('\n'); err != nil {
				if err == io.EOF {
					err = nil
				}
				return
			}
		}
		if filepath.IsAbs(line) || isURL(line) {
			track.Path = line
		} else {
			track.Path = filepath.Join(pl.dest, line)
		}
		pl.Tracks = append(pl.Tracks, track)
		n++
	}
}

func (pl *Playlist) WriteTo(w io.Writer) (n int64, err error) {
	if _, err = fmt.Fprintln(w, "#EXTM3U"); err != nil {
		return
	}
	const sep = string(filepath.Separator)
	var dir0 = filepath.Clean(pl.dest)
	var dir1 = filepath.Dir(dir0)
	dir0 += sep
	dir1 += sep
	for _, track := range pl.Tracks {
		if track.Time > 0 {
			if _, err = fmt.Fprintf(w, "#EXTINF:%d,%s\n", track.Time, track.Name); err != nil {
				return
			}
		}
		var fpath = track.Path
		if strings.HasPrefix(fpath, dir0) || strings.HasPrefix(fpath, dir1) {
			if fpath, err = filepath.Rel(dir0, fpath); err != nil {
				return
			}
		}
		if _, err = fmt.Fprintln(w, fpath); err != nil {
			return
		}
		n++
	}
	return
}

func (pl *Playlist) Close() error {
	return nil
}

func (pl *Playlist) OpenFile(fpath string) (r VFile, err error) {
	/*if isURL(fpath) {
		var resp *http.Response
		resp, err = http.Get(fpath)
		r = resp.Body
		return
	}*/
	if r, err = os.Open(fpath); err == nil {
		return
	}
	return
}

// The End.
