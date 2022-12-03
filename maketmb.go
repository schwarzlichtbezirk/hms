package hms

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/gift"

	_ "github.com/chai2010/webp"     // register WebP
	_ "github.com/oov/psd"           // register PSD format
	_ "github.com/spate/glimage/dds" // register DDS format
	_ "golang.org/x/image/bmp"       // register BMP format
	_ "golang.org/x/image/tiff"      // register TIFF format

	_ "github.com/ftrvxmtrx/tga" // put TGA to end, decoder does not register magic prefix
)

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

// Encoder configures encoding PNG images for thumbnails and tiles.
var tmbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// Error messages
var (
	ErrBadMedia = errors.New("media content is corrupted")
	ErrNoThumb  = errors.New("music file without thumbnail")
	ErrNotFile  = errors.New("property is not file")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
	ErrImgNil   = errors.New("can not allocate image")
)

// ExtractTmb extract thumbnail from embedded file tags.
func ExtractTmb(syspath string) (md *MediaData, err error) {
	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		return
	}

	// try to extract from EXIF
	if ek, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		if ek.thumb.Mime != MimeDis {
			md = &ek.thumb
			return // thumbnail from EXIF
		}
	}

	// try to extract from ID3
	if tk, ok := prop.(*TagKit); ok { // skip non-ID3 properties
		if tk.thumb.Mime != MimeDis {
			md = &tk.thumb
			return // thumbnail from tags
		}
	}
	return
}

// MakeThumb produces new thumbnail object.
func MakeThumb(r io.Reader, orientation int) (md *MediaData, err error) {
	// create sized image for thumbnail
	var ftype string
	var src, dst image.Image
	if src, ftype, err = image.Decode(r); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}
	if src.Bounds().In(image.Rect(0, 0, cfg.TmbResolution[0], cfg.TmbResolution[1])) {
		dst = src
	} else {
		var fltlst = AddOrientFilter([]gift.Filter{
			gift.ResizeToFit(cfg.TmbResolution[0], cfg.TmbResolution[1], gift.LinearResampling),
		}, orientation)
		var thumbfilter = gift.New(fltlst...)
		var img = image.NewRGBA(thumbfilter.Bounds(src.Bounds()))
		if img.Pix == nil {
			err = ErrImgNil
			return // out of memory
		}
		thumbfilter.Draw(img, src)
		dst = img
	}

	// create valid thumbnail
	return ToNativeImg(dst, ftype)
}

// ToNativeImg converts Image to specified file format supported by browser.
func ToNativeImg(m image.Image, ftype string) (md *MediaData, err error) {
	var buf bytes.Buffer
	var mime Mime_t
	switch ftype {
	case "gif":
		if err = gif.Encode(&buf, m, nil); err != nil {
			return // can not write gif
		}
		mime = MimeGif
	case "png", "dds", "webp", "psd":
		if err = tmbpngenc.Encode(&buf, m); err != nil {
			return // can not write png
		}
		mime = MimePng
	default:
		if err = jpeg.Encode(&buf, m, &jpeg.Options{Quality: cfg.TmbJpegQuality}); err != nil {
			return // can not write jpeg
		}
		mime = MimeJpeg
	}
	md = &MediaData{
		Data: buf.Bytes(),
		Mime: mime,
	}
	return
}

// GetCachedThumb tries to extract existing thumbnail from cache, otherwise
// makes new one and put it to cache.
func GetCachedThumb(syspath string) (md *MediaData, err error) {
	// try to extract thumbnail from package
	if md, err = thumbpkg.GetImage(syspath); err != nil || md != nil {
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		return
	}

	if cfg.FitEmbeddedTmb {
		if tk, ok := prop.(*TagKit); ok { // skip non-ID3 properties
			if tk.thumb.Data == nil {
				err = ErrNoThumb
				return // music file without thumbnail
			}
			if md, err = MakeThumb(bytes.NewReader(tk.thumb.Data), OrientNormal); err != nil {
				return
			}
			// push thumbnail to package
			err = thumbpkg.PutImage(syspath, md)
			return
		}
	}

	// check that file is image
	if !IsTypeImage(GetFileExt(syspath)) {
		err = ErrNotImg
		return // file is not image
	}

	if prop.(Pather).Size() > cfg.ThumbFileMaxSize {
		err = ErrTooBig
		return // file is too big
	}

	var r io.ReadCloser
	if r, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer r.Close()

	// try to extract orientation from EXIF
	var orientation = OrientNormal
	if ek, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		orientation = ek.Orientation
	}

	if md, err = MakeThumb(r, orientation); err != nil {
		return
	}

	// push thumbnail to package
	err = thumbpkg.PutImage(syspath, md)
	return
}

// MakeTile produces new tile object.
func MakeTile(r io.Reader, wdh, hgt int, orientation int) (md *MediaData, err error) {
	var ftype string
	var src, dst image.Image
	if src, ftype, err = image.Decode(r); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}

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

	return ToNativeImg(dst, ftype)
}

// GetCachedTile tries to extract existing tile from cache, otherwise
// makes new one and put it to cache.
func GetCachedTile(syspath string, wdh, hgt int) (md *MediaData, err error) {
	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)

	// try to extract tile from package
	if md, err = tilespkg.GetImage(tilepath); err != nil || md != nil {
		return
	}

	var prop interface{}
	if prop, err = propcache.Get(syspath); err != nil {
		return // can not get properties
	}

	// check that file is image
	if !IsTypeImage(GetFileExt(syspath)) {
		err = ErrNotImg
		return // file is not image
	}

	if prop.(Pather).Size() > cfg.ThumbFileMaxSize {
		err = ErrTooBig
		return // file is too big
	}

	var r io.ReadCloser
	if r, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer r.Close()

	// try to extract orientation from EXIF
	var orientation = OrientNormal
	if ek, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		orientation = ek.Orientation
	}

	if md, err = MakeTile(r, wdh, hgt, orientation); err != nil {
		return
	}

	// push tile to package
	err = tilespkg.PutImage(tilepath, md)
	return
}

// The End.
