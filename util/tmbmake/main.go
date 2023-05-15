package main

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/gift"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

var root = "D:/test/imggps"

// GetFileExt returns file extension converted to lowercase.
func GetFileExt(fname string) string {
	return strings.ToLower(path.Ext(fname))
}

// IsTypeImage checks that file is some image format.
func IsTypeImage(ext string) bool {
	switch ext {
	case ".tga", ".bmp", ".dib", ".rle", ".dds", ".tif", ".tiff",
		".jpg", ".jpe", ".jpeg", ".jfif", ".gif", ".png", ".webp", ".avif",
		".psd", ".psb", ".jp2", ".jpg2", ".jpx", ".jpm", ".jxr":
		return true
	}
	return false
}

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

type Mime_t int16

const (
	MimeDis  Mime_t = -1 // file can not be cached for thumbnails.
	MimeNil  Mime_t = 0  // file is not cached for thumbnails, have indeterminate state.
	MimeUnk  Mime_t = 1  // image/*
	MimeGif  Mime_t = 2  // image/gif
	MimePng  Mime_t = 3  // image/png
	MimeJpeg Mime_t = 4  // image/jpeg
	MimeWebp Mime_t = 5  // image/webp
)

var MimeStr = map[Mime_t]string{
	MimeNil:  "",
	MimeUnk:  "image/*",
	MimeGif:  "image/gif",
	MimePng:  "image/png",
	MimeJpeg: "image/jpeg",
	MimeWebp: "image/webp",
}

var MimeVal = map[string]Mime_t{
	"image/*":    MimeUnk,
	"image/gif":  MimeGif,
	"image/png":  MimePng,
	"image/jpg":  MimeJpeg,
	"image/jpeg": MimeJpeg,
	"image/webp": MimeWebp,
}

var MimeExt = map[string]Mime_t{
	"gif":  MimeGif,
	"png":  MimePng,
	"jpg":  MimeJpeg,
	"jpeg": MimeJpeg,
	"webp": MimeWebp,
}

func GetMimeVal(mime, ext string) Mime_t {
	if mime, ok := MimeVal[mime]; ok {
		return mime
	}
	if mime, ok := MimeExt[strings.ToLower(ext)]; ok {
		return mime
	}
	if mime, ok := MimeExt[strings.ToLower(mime)]; ok {
		return mime
	}
	return MimeUnk
}

// MediaData is thumbnails cache element.
type MediaData struct {
	Data []byte
	Mime Mime_t
}

// EXIF image orientation constants.
const (
	// orientation: normal
	OrientNormal = 1
	// orientation: horizontal reversed
	OrientHorzReversed = 2
	// orientation: flipped
	OrientFlipped = 3
	// orientation: flipped & horizontal reversed
	OrientFlipHorzReversed = 4
	// orientation: clockwise turned & horizontal reversed
	OrientCwHorzReversed = 5
	// orientation: clockwise turned
	OrientCw = 6
	// orientation: anticlockwise turned & horizontal reversed
	OrientAcwHorzReversed = 7
	// orientation: anticlockwise turned
	OrientAcw = 8
)

// AddOrientFilter appends filters to bring image to normal orientation.
func AddOrientFilter(flt []gift.Filter, orientation int) []gift.Filter {
	switch orientation {
	case OrientHorzReversed: // orientation: horizontal reversed
		flt = append(flt, gift.FlipHorizontal())
	case OrientFlipped: // orientation: flipped
		flt = append(flt, gift.Rotate180())
	case OrientFlipHorzReversed: // orientation: flipped & horizontal reversed
		flt = append(flt, gift.Rotate180())
		flt = append(flt, gift.FlipHorizontal())
	case OrientCwHorzReversed: // orientation: clockwise turned & horizontal reversed
		flt = append(flt, gift.Rotate270())
		flt = append(flt, gift.FlipHorizontal())
	case OrientCw: // clockwise turned
		flt = append(flt, gift.Rotate270())
	case OrientAcwHorzReversed: // orientation: anticlockwise turned & horizontal reversed
		flt = append(flt, gift.Rotate90())
		flt = append(flt, gift.FlipHorizontal())
	case OrientAcw: // anticlockwise turned
		flt = append(flt, gift.Rotate90())
	}
	return flt
}

// Error messages
var (
	ErrBadMedia = errors.New("media content is corrupted")
	ErrNoThumb  = errors.New("embedded thumbnail is not found")
	ErrNotFile  = errors.New("property is not file")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
	ErrImgNil   = errors.New("can not allocate image")
)

// DrawTile produces new tile object.
func DrawTile(src image.Image, wdh, hgt int, orientation int) (md MediaData, err error) {
	var dst image.Image
	switch orientation {
	case OrientCwHorzReversed, OrientCw, OrientAcwHorzReversed, OrientAcw:
		wdh, hgt = hgt, wdh
	}
	var fltlst = AddOrientFilter([]gift.Filter{
		gift.ResizeToFill(wdh, hgt, gift.LinearResampling, gift.CenterAnchor),
	}, orientation)
	var filter = gift.New(fltlst...)
	var img = image.NewRGBA(filter.Bounds(src.Bounds()))
	if img.Pix == nil {
		err = ErrImgNil
		return // out of memory
	}
	filter.Draw(img, src)
	dst = img

	return EncodeRGBA2WebP(dst)
}

// EncodeRGBA2WebP converts Image to WebP file lossless format with alpha channel.
func EncodeRGBA2WebP(m image.Image) (md MediaData, err error) {
	var data []byte
	if data, err = webp.EncodeRGBA(m, 80 /*cfg.TmbWebpQuality*/); err != nil {
		return // can not write webp
	}
	md.Data = data
	md.Mime = MimeWebp
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

	if fi.Size() > 50*1024*1024 /*cfg.ThumbFileMaxSize*/ {
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
		var md MediaData
		if md, err = DrawTile(src, wdh, hgt, orientation); err != nil {
			return
		}
		_, _ = tilepath, md
	}

	return
}

func main() {
	var err error
	var list []string
	if list, err = FileList(os.DirFS(root)); err != nil {
		log.Fatal(err)
	}
	for i, fpath := range list {
		fmt.Fprintln(os.Stdout, i, fpath)
	}
}

// The End.
