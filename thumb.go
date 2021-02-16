package hms

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"

	"github.com/disintegration/gift"
	_ "github.com/oov/psd"           // register PSD format
	_ "github.com/spate/glimage/dds" // register DDS format
	_ "golang.org/x/image/bmp"       // register BMP format
	_ "golang.org/x/image/tiff"      // register TIFF format
	_ "golang.org/x/image/webp"      // register WebP format

	_ "github.com/ftrvxmtrx/tga" // put TGA to end, decoder does not register magic prefix
)

const (
	// TMBnone - file is not cached for thumbnails.
	TMBnone = 0
	// TMBreject - file can not be cached for thumbnails.
	TMBreject = -1
	// TMBcached - file is already cached for thumbnails.
	TMBcached = 1
)

// Allow images a bit larger than standard icon stay as is.
var thumbmaxrect = image.Rect(0, 0, 320, 320)

// Resize big images to fit into standard icon size.
var thumbfilter = gift.New(
	gift.ResizeToFit(256, 256, gift.LinearResampling),
)

var thumbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// Error messages
var (
	ErrBadMedia = errors.New("media content is corrupted")
	ErrNotThumb = errors.New("thumbnail content can not be created")
	ErrNotFile  = errors.New("property is not file")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
)

// MediaData is thumbnails cache element.
type MediaData struct {
	Data []byte
	Mime string
}

// TmbProp is thumbnails properties.
type TmbProp struct {
	PUIDVal string `json:"puid,omitempty" yaml:"puid,omitempty"`
	NTmbVal int    `json:"ntmb,omitempty" yaml:"ntmb,omitempty"`
}

// Setup generates PUID (path unique identifier) and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.PUIDVal = pathcache.Cache(syspath)
	tp.NTmbCached()
}

// NTmbCached updates cached state for this cache key.
func (tp *TmbProp) NTmbCached() {
	if thumbcache.Has(tp.PUIDVal) {
		tp.NTmbVal = TMBcached
	} else {
		tp.NTmbVal = TMBnone
	}
}

// PUID returns thumbnail key, it's full system path unique ID.
func (tp *TmbProp) PUID() string {
	return tp.PUIDVal
}

// NTmb returns thumbnail state, -1 impossible, 0 undefined, 1 ready.
func (tp *TmbProp) NTmb() int {
	return tp.NTmbVal
}

// SetNTmb updates thumbnail state to given value.
func (tp *TmbProp) SetNTmb(v int) {
	tp.NTmbVal = v
}

// FindTmb finds thumbnail in embedded file tags, or build it if it possible.
func FindTmb(prop Pather, syspath string) (md *MediaData, err error) {
	if prop.Type() < 0 {
		err = ErrNotFile
		return // not a file
	}

	// try to extract from EXIF
	if _, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		if md, err = GetExifTmb(syspath); err == nil {
			return // thumbnail from EXIF
		}
	}

	// try to extract from ID3
	if _, ok := prop.(*TagKit); ok { // skip non-ID3 properties
		if md, err = GetTagTmb(syspath); err == nil {
			return // thumbnail from ID3
		}
	}

	if prop.Size() > cfg.ThumbFileMaxSize {
		err = ErrTooBig
		return // file is too big
	}

	// check all others are images
	if typetogroup[prop.Type()] != FGimage {
		err = ErrNotImg
		return // file is not image
	}

	var file *os.File
	if file, err = os.Open(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	return MakeTmb(file)
}

// MakeTmb reads image from the stream and makes thumbnail with format
// depended from alpha-channel is present in the original image.
func MakeTmb(r io.Reader) (md *MediaData, err error) {
	var ftype string
	var src, dst image.Image
	if src, ftype, err = image.Decode(r); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
		}
	}
	if src.Bounds().In(thumbmaxrect) {
		dst = src
	} else {
		var img = image.NewRGBA(thumbfilter.Bounds(src.Bounds()))
		thumbfilter.Draw(img, src)
		dst = img
	}

	var buf bytes.Buffer
	var mime string
	switch ftype {
	case "gif":
		if err = gif.Encode(&buf, dst, nil); err != nil {
			return // can not write gif
		}
		mime = "image/gif"
	case "png", "dds", "webp", "psd":
		if err = thumbpngenc.Encode(&buf, dst); err != nil {
			return // can not write png
		}
		mime = "image/png"
	default:
		if err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
			return // can not write jpeg
		}
		mime = "image/jpeg"
	}
	md = &MediaData{
		Data: buf.Bytes(),
		Mime: mime,
	}
	return // set valid thumbnail
}

// APIHANDLER
func tmbchkAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Tmbs []*TmbProp `json:"tmbs"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.Tmbs) == 0 {
		WriteError400(w, ErrNoData, AECtmbchknodata)
		return
	}

	for _, tp := range arg.Tmbs {
		if syspath, ok := pathcache.Path(tp.PUID()); ok {
			if prop, err := propcache.Get(syspath); err == nil {
				tp.NTmbVal = prop.(Pather).NTmb()
			}
		}
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID   int      `json:"aid"`
		PUIDs []string `json:"puids"`
	}

	// get arguments
	if err = AjaxGetArg(r, &arg); err != nil {
		WriteJSON(w, http.StatusBadRequest, err)
		return
	}
	if len(arg.PUIDs) == 0 {
		WriteError400(w, ErrNoData, AECtmbscnnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(int(arg.AID)); prf == nil {
		WriteError400(w, ErrNoAcc, AECtmbscnnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(r); err != nil {
		WriteJSON(w, http.StatusUnauthorized, err)
		return
	}

	go func() {
		for _, puid := range arg.PUIDs {
			if syspath, ok := pathcache.Path(puid); ok {
				if cg := prf.PathAccess(syspath, auth == prf); cg.IsZero() {
					continue
				}
				thumbcache.Get(puid)
			}
		}
	}()

	WriteOK(w, nil)
}

// The End.
