package main

import (
	"fmt"
	"image"
	"io/fs"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	. "github.com/schwarzlichtbezirk/hms"
)

func FileList(fsys fs.FS) (list []string, err error) {
	fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && IsTypeImage(GetFileExt(fpath)) {
			list = append(list, fpath)
		}
		return nil
	})
	return
}

// Tiles multipliers:
var tilemult = [...]int{
	2, 3, 4, 6, 8, 9, 10, 12, 15, 16, 18, 20, 24, 30, 36,
}

func Convert(fpath string) (err error) {
	var r *os.File
	if r, err = os.Open(fpath); err != nil {
		return // can not open file
	}
	defer r.Close()

	var fi fs.FileInfo
	if fi, err = r.Stat(); err != nil {
		return
	}
	if fi.IsDir() {
		err = ErrNotFile // file is directory
		return
	}

	var ext = GetFileExt(fpath)

	// check that file is image
	if !IsTypeImage(ext) {
		err = ErrNotImg
		return // file is not image
	}

	if fi.Size() > 50*1024*1024 /*Cfg.ThumbFileMaxSize*/ {
		err = ErrTooBig
		return // file is too big
	}

	var x *exif.Exif
	if x, err = exif.Decode(r); err != nil {
		return
	}

	var orientation = OrientNormal
	var t *tiff.Tag
	if t, err = x.Get(exif.Orientation); err == nil {
		orientation, _ = t.Int(0)
	}

	var src image.Image
	if src, _, err = image.Decode(r); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}
	for _, tm := range tilemult {
		var wdh, hgt = tm * 24, tm * 18
		var tilepath = fmt.Sprintf("%s?%dx%d", fpath, wdh, hgt)
		var b []byte
		if b, err = DrawTile(src, wdh, hgt, orientation); err != nil {
			return
		}
		_, _ = tilepath, b
	}

	return
}

// The End.
