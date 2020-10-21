package hms

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/disintegration/gift"
	_ "github.com/oov/psd"
	_ "github.com/spate/glimage/dds"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	_ "github.com/ftrvxmtrx/tga" // put TGA to end, decoder does not register magic prefix
)

const (
	TMB_none   = 0
	TMB_reject = -1
	TMB_cached = 1
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

// HTTP error messages
var (
	ErrNoHash   = errors.New("file with given hash not found")
	ErrBadThumb = errors.New("thumbnail cache is corrupted")
	ErrNotThumb = errors.New("thumbnail content can not be created")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
)

// Thumbnails cache element.
type ThumbElem struct {
	Data       []byte
	Mime       string
	OrgW, OrgH int
	TmbW, TmbH int
}

// Thumbnails properties.
type TmbProp struct {
	HashVal string `json:"hash,omitempty"`
	NTmbVal int    `json:"ntmb,omitempty"`
}

// Generates cache key as hash of path and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.HashVal = hashcache.Cache(syspath)
	tp.NTmbCached()
}

// Updates cached state for this cache key.
func (tp *TmbProp) NTmbCached() {
	if thumbcache.Has(tp.HashVal) {
		tp.NTmbVal = TMB_cached
	} else {
		tp.NTmbVal = TMB_none
	}
}

// Thumbnail key, it's MD5-hash of full path.
func (tp *TmbProp) Hash() string {
	return tp.HashVal
}

// Thumbnail state, -1 impossible, 0 undefined, 1 ready.
func (tp *TmbProp) NTmb() int {
	return tp.NTmbVal
}

// Updates thumbnail state to given value.
func (tp *TmbProp) SetNTmb(v int) {
	tp.NTmbVal = v
}

func FindTmb(prop Proper, syspath string) (tmb *ThumbElem, err error) {
	// try to extract from EXIF
	if _, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		if tmb, err = GetExifTmb(syspath); err == nil {
			return // thumbnail from EXIF
		}
	}

	// try to extract from ID3
	if _, ok := prop.(*TagKit); ok { // skip non-ID3 properties
		if tmb, err = GetTagTmb(syspath); err == nil {
			return // thumbnail from ID3
		}
	}

	if prop.Size() > cfg.ThumbFileMaxSize {
		err = ErrTooBig
		return // file is too big
	}

	// check all others are images
	if typetogroup[prop.Type()] != FG_image {
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

func MakeTmb(r io.Reader) (tmb *ThumbElem, err error) {
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
	tmb = &ThumbElem{
		Data: buf.Bytes(),
		Mime: mime,
		OrgW: src.Bounds().Dx(),
		OrgH: src.Bounds().Dy(),
		TmbW: dst.Bounds().Dx(),
		TmbH: dst.Bounds().Dy(),
	}
	return // set valid thumbnail
}

// APIHANDLER
func tmbchkApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Tmbs []*TmbProp `json:"tmbs"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_tmbchkbadreq)
			return
		}
		if len(arg.Tmbs) == 0 {
			WriteError400(w, ErrNoData, EC_tmbchknodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_tmbchknoreq)
		return
	}

	for _, tp := range arg.Tmbs {
		if syspath, ok := hashcache.Path(tp.Hash()); ok {
			if prop, err := propcache.Get(syspath); err == nil {
				tp.NTmbVal = prop.(Proper).NTmb()
			}
		}
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID   int      `json:"aid"`
		Paths []string `json:"paths"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_tmbscnbadreq)
			return
		}
		if len(arg.Paths) == 0 {
			WriteError400(w, ErrNoData, EC_tmbscnnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_tmbscnnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_tmbscnnoacc)
		return
	}

	go func() {
		for _, shrpath := range arg.Paths {
			var syspath = acc.GetSharePath(shrpath)
			if hash, ok := hashcache.Key(syspath); ok {
				thumbcache.Get(hash)
			}
		}
	}()

	WriteOK(w, nil)
}

// The End.
