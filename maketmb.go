package hms

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"strings"
	"time"

	. "github.com/schwarzlichtbezirk/hms/config"
	. "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/disintegration/gift"

	"github.com/chai2010/webp"       // register WebP
	_ "github.com/oov/psd"           // register PSD format
	_ "github.com/spate/glimage/dds" // register DDS format
	_ "golang.org/x/image/bmp"       // register BMP format
	_ "golang.org/x/image/tiff"      // register TIFF format

	_ "github.com/ftrvxmtrx/tga" // put TGA to end, decoder does not register magic prefix
)

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
	Time time.Time
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

// CheckImageSize compares given image size with maximum for image type
// (plain bitmap or compresed image).
func CheckImageSize(ext string, size int64) bool {
	switch ext {
	case ".tga", ".bmp", ".dib", ".rle", ".dds",
		".tif", ".tiff", ".psd", ".psb":
		return size < Cfg.BitmapMaxSize || Cfg.BitmapMaxSize == 0
	case ".jpg", ".jpe", ".jpeg", ".jfif",
		".jp2", ".jpg2", ".jpx", ".jpm", ".jxr",
		".gif", ".png", ".webp", ".avif":
		return size < Cfg.JpegMaxSize || Cfg.JpegMaxSize == 0
	}
	return false
}

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
		tmbcache.Poke(puid, md)
		tmbcache.ToLimit(Cfg.ThumbCacheMaxNum)
	}
	return
}

// DrawThumb produces new thumbnail object.
func DrawThumb(src image.Image, orientation int) (data []byte, err error) {
	var dst image.Image
	if src.Bounds().In(image.Rect(0, 0, Cfg.TmbResolution[0], Cfg.TmbResolution[1])) {
		dst = src
	} else {
		var fltlst = AddOrientFilter([]gift.Filter{
			gift.ResizeToFit(Cfg.TmbResolution[0], Cfg.TmbResolution[1], gift.LinearResampling),
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
	if data, err = webp.EncodeRGBA(dst, Cfg.TmbWebpQuality); err != nil {
		return // can not write webp
	}
	return
}

// CacheThumb tries to extract existing thumbnail from cache, otherwise
// makes new one and put it to cache.
func CacheThumb(session *Session, syspath string) (md MediaData, err error) {
	// try to extract thumbnail from package
	if md, err = ThumbPkg.GetData(syspath); err != nil {
		return // failure
	}
	if md.Data != nil {
		return // extracted
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
		if Cfg.FitEmbeddedTmb {
			var mdtag MediaData
			if mdtag, err = ExtractThumbID3(syspath); err != nil {
				return
			}
			// create sized image for thumbnail
			var src image.Image
			if src, _, err = image.Decode(bytes.NewReader(mdtag.Data)); err != nil {
				if src == nil { // skip "short Huffman data" or others errors with partial results
					return // can not decode file by any codec
				}
			}
			if md.Data, err = DrawThumb(src, OrientNormal); err != nil {
				return
			}
			md.Mime = MimeWebp
			md.Time = mdtag.Time
			// push thumbnail to package
			err = ThumbPkg.PutFile(syspath, md)
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

	if !CheckImageSize(ext, fi.Size()) {
		err = ErrTooBig
		return // file is too big
	}

	// try to extract orientation from EXIF
	var orientation = OrientNormal
	var puid = PathStoreCache(session, syspath)
	if ep, ok := ExifStoreGet(session, puid); ok { // skip non-EXIF properties
		if ep.Orientation > 0 {
			orientation = ep.Orientation
		}
	}

	var file File
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	// create sized image for thumbnail
	var src image.Image
	if src, _, err = image.Decode(file); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}
	if md.Data, err = DrawThumb(src, orientation); err != nil {
		return
	}
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}

	// push thumbnail to package
	err = ThumbPkg.PutFile(syspath, md)
	return
}

// DrawTile produces new tile object.
func DrawTile(src image.Image, wdh, hgt int, orientation int) (data []byte, err error) {
	switch orientation {
	case OrientCwHorzReversed, OrientCw, OrientAcwHorzReversed, OrientAcw:
		wdh, hgt = hgt, wdh
	}
	var fltlst = AddOrientFilter([]gift.Filter{
		gift.ResizeToFill(wdh, hgt, gift.LinearResampling, gift.CenterAnchor),
	}, orientation)
	var filter = gift.New(fltlst...)
	var dst = image.NewRGBA(filter.Bounds(src.Bounds()))
	if dst.Pix == nil {
		err = ErrImgNil
		return // out of memory
	}
	filter.Draw(dst, src)

	if data, err = webp.EncodeRGBA(dst, Cfg.TmbWebpQuality); err != nil {
		return // can not write webp
	}
	return
}

// CacheTile tries to extract existing tile from cache, otherwise
// makes new one and put it to cache.
func CacheTile(session *Session, syspath string, wdh, hgt int) (md MediaData, err error) {
	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, wdh, hgt)

	// try to extract tile from package
	if md, err = TilesPkg.GetData(tilepath); err != nil {
		return // failure
	}
	if md.Data != nil {
		return // extracted
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

	if !CheckImageSize(ext, fi.Size()) {
		err = ErrTooBig
		return // file is too big
	}

	var file File
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	// try to extract orientation from EXIF
	var orientation = OrientNormal
	var puid = PathStoreCache(session, syspath)
	if ep, ok := ExifStoreGet(session, puid); ok { // skip non-EXIF properties
		if ep.Orientation > 0 {
			orientation = ep.Orientation
		}
	}

	var src image.Image
	if src, _, err = image.Decode(file); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}
	if md.Data, err = DrawTile(src, wdh, hgt, orientation); err != nil {
		return
	}
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}

	// push tile to package
	err = TilesPkg.PutFile(tilepath, md)
	return
}

// The End.
