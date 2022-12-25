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
	"io/fs"

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
	ErrNoThumb  = errors.New("embedded thumbnail is not found")
	ErrNotFile  = errors.New("property is not file")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
	ErrImgNil   = errors.New("can not allocate image")
)

// ExtractThmub extract thumbnail from embedded file tags.
func ExtractThmub(session *Session, syspath string) (md MediaData, err error) {
	var puid = PathStoreCache(session, syspath)
	var ok bool
	if md, ok = tmbcache.Peek(puid); ok {
		return
	}

	var ext = GetFileExt(syspath)
	if IsTypeID3(ext) {
		md, err = ExtractThumbID3(syspath)
	} else if IsTypeEXIF(ext) {
		md, err = ExtractThumbEXIF(syspath)
	} else {
		md.Mime = MimeDis
		err = ErrNoThumb
	}

	// push successful result to cache, err != nil, md.Mime != MimeDis
	if err != nil {
		tmbcache.Push(puid, md)
		tmbcache.ToLimit(cfg.ThumbCacheMaxNum)
	}
	return
}

// MakeThumb produces new thumbnail object.
func MakeThumb(r io.Reader, orientation int) (md MediaData, err error) {
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
func ToNativeImg(m image.Image, ftype string) (md MediaData, err error) {
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
	md.Data = buf.Bytes()
	md.Mime = mime
	return
}

// CacheThumb tries to extract existing thumbnail from cache, otherwise
// makes new one and put it to cache.
func CacheThumb(session *Session, syspath string) (md MediaData, err error) {
	// try to extract thumbnail from package
	if md, err = thumbpkg.GetImage(syspath); err != nil || md.Mime != MimeNil {
		return
	}

	var fi fs.FileInfo
	if fi, err = StatFile(syspath); err != nil {
		return
	}
	if fi.IsDir() {
		err = ErrNotFile // file is directory
		return
	}

	var ext = GetFileExt(syspath)
	if IsTypeID3(ext) {
		if cfg.FitEmbeddedTmb {
			var mdtag MediaData
			if mdtag, err = ExtractThumbID3(syspath); err != nil {
				return
			}
			if md, err = MakeThumb(bytes.NewReader(mdtag.Data), OrientNormal); err != nil {
				return
			}
			// push thumbnail to package
			err = thumbpkg.PutImage(syspath, md)
			return
		} else {
			err = ErrNotImg
			return // file is not image
		}
	}

	// check that file is image
	if !IsTypeImage(ext) {
		err = ErrNotImg
		return // file is not image
	}

	if fi.Size() > cfg.ThumbFileMaxSize {
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
	var puid = PathStoreCache(session, syspath)
	if ep, ok := ExifStoreGet(session, puid); ok { // skip non-EXIF properties
		if ep.Orientation > 0 {
			orientation = ep.Orientation
		}
	}

	if md, err = MakeThumb(r, orientation); err != nil {
		return
	}

	// push thumbnail to package
	err = thumbpkg.PutImage(syspath, md)
	return
}

// MakeTile produces new tile object.
func MakeTile(r io.Reader, wdh, hgt int, orientation int) (md MediaData, err error) {
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

// CacheTile tries to extract existing tile from cache, otherwise
// makes new one and put it to cache.
func CacheTile(session *Session, syspath string, wdh, hgt int) (md MediaData, err error) {
	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)

	// try to extract tile from package
	if md, err = tilespkg.GetImage(tilepath); err != nil || md.Mime != MimeNil {
		return
	}

	var fi fs.FileInfo
	if fi, err = StatFile(syspath); err != nil {
		return
	}
	if fi.IsDir() {
		err = ErrNotFile // file is directory
		return
	}

	var ext = GetFileExt(syspath)

	// check that file is image
	if !IsTypeImage(ext) {
		err = ErrNotImg
		return // file is not image
	}

	if fi.Size() > cfg.ThumbFileMaxSize {
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
	var puid = PathStoreCache(session, syspath)
	if ep, ok := ExifStoreGet(session, puid); ok { // skip non-EXIF properties
		if ep.Orientation > 0 {
			orientation = ep.Orientation
		}
	}

	if md, err = MakeTile(r, wdh, hgt, orientation); err != nil {
		return
	}

	// push tile to package
	err = tilespkg.PutImage(tilepath, md)
	return
}

// The End.
