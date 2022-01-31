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

// Encoder configures encoding PNG images for thumbnails and tiles.
var tmbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// Error messages
var (
	ErrBadMedia = errors.New("media content is corrupted")
	ErrNotThumb = errors.New("thumbnail content can not be created")
	ErrNotFile  = errors.New("property is not file")
	ErrNotImg   = errors.New("file is not image")
	ErrTooBig   = errors.New("file is too big")
	ErrImgNil   = errors.New("can not allocate image")
)

// MediaData is thumbnails cache element.
type MediaData struct {
	Data []byte
	Mime string
}

// TmbProp is thumbnails properties.
type TmbProp struct {
	PUIDVal PuidType `json:"puid,omitempty" yaml:"puid,omitempty"`
	NTmbVal int      `json:"ntmb,omitempty" yaml:"ntmb,omitempty"`
	MTmbVal string   `json:"mtmb,omitempty" yaml:"mtmb,omitempty"`
}

// Setup generates PUID (path unique identifier) and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.PUIDVal = syspathcache.Cache(syspath)
	tp.UpdateTmb()
}

// UpdateTmb updates cached state for this cache key.
func (tp *TmbProp) UpdateTmb() {
	if thumbcache.Has(tp.PUIDVal) {
		var v, err = thumbcache.Get(tp.PUIDVal)
		if err != nil {
			tp.SetTmb(TMBreject, "")
			return
		}
		var md, ok = v.(*MediaData)
		if !ok {
			tp.SetTmb(TMBreject, "")
			return
		}
		tp.SetTmb(TMBcached, md.Mime)
	} else {
		tp.SetTmb(TMBnone, "")
		return
	}
}

// PUID returns thumbnail key, it's full system path unique ID.
func (tp *TmbProp) PUID() PuidType {
	return tp.PUIDVal
}

// NTmb returns thumbnail state, -1 impossible, 0 undefined, 1 ready.
func (tp *TmbProp) NTmb() int {
	return tp.NTmbVal
}

// MTmb returns thumbnail MIME type, if thumbnail is present and NTmb is 1.
func (tp *TmbProp) MTmb() string {
	return tp.MTmbVal
}

// SetTmb updates thumbnail state to given value.
func (tp *TmbProp) SetTmb(tmb int, mime string) {
	tp.NTmbVal = tmb
	tp.MTmbVal = mime
}

// FindTmb finds thumbnail in embedded file tags, or build it if it possible.
func FindTmb(prop Pather, syspath string) (md *MediaData, err error) {
	if prop.Type() < 0 {
		err = ErrNotFile
		return // not a file
	}

	// try to extract from EXIF
	if cfg.UseEmbeddedTmb {
		if _, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
			if md, err = GetExifTmb(syspath); err == nil {
				return // thumbnail from EXIF
			}
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
	if GetFileGroup(prop.Name()) != FGimage {
		err = ErrNotImg
		return // file is not image
	}

	var file VFile
	if file, err = OpenFile(syspath); err != nil {
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
	if src.Bounds().In(image.Rect(0, 0, 320, 320)) {
		dst = src
	} else {
		var thumbfilter = gift.New(
			gift.ResizeToFit(cfg.TmbResolution[0], cfg.TmbResolution[1], gift.LinearResampling),
		)
		var img = image.NewRGBA(thumbfilter.Bounds(src.Bounds()))
		if img.Pix == nil {
			err = ErrImgNil
			return // out of memory
		}
		thumbfilter.Draw(img, src)
		dst = img
	}

	return ToNativeImg(dst, ftype) // set valid thumbnail
}

// ToNativeImg converts Image to specified file format supported by browser.
func ToNativeImg(m image.Image, ftype string) (md *MediaData, err error) {
	var buf bytes.Buffer
	var mime string
	switch ftype {
	case "gif":
		if err = gif.Encode(&buf, m, nil); err != nil {
			return // can not write gif
		}
		mime = "image/gif"
	case "png", "dds", "webp", "psd":
		if err = tmbpngenc.Encode(&buf, m); err != nil {
			return // can not write png
		}
		mime = "image/png"
	default:
		if err = jpeg.Encode(&buf, m, &jpeg.Options{Quality: cfg.TmbJpegQuality}); err != nil {
			return // can not write jpeg
		}
		mime = "image/jpeg"
	}
	md = &MediaData{
		Data: buf.Bytes(),
		Mime: mime,
	}
	return
}

// ThumbScanner is singleton for thumbnails producing
// with single queue to prevent overload.
var ThumbScanner scanner

type scanner struct {
	put chan PuidType
	del chan PuidType
}

// Scan is goroutine for thumbnails scanning.
func (s *scanner) Scan() {
	s.put = make(chan PuidType)
	s.del = make(chan PuidType)

	var list []PuidType
	var ctx chan struct{}

	var cache = func() {
		if len(list) > 0 {
			var puid = list[0]
			list = list[1:]
			ctx = make(chan struct{})
			go func() {
				defer close(ctx)
				if puid != 0 {
					thumbcache.Get(puid)
				}
			}()
		}
	}

	for {
		select {
		case puid := <-s.put:
			list = append(list, puid)
			if ctx == nil {
				cache()
			}
		case puid1 := <-s.del:
			for i, puid2 := range list {
				if puid1 == puid2 {
					list = append(list[:i], list[i+1:]...)
					break
				}
			}
		case <-ctx:
			ctx = nil
			cache()
		}
	}
}

// Add list of PUIDs to queue to make thumbnails.
func (s *scanner) Add(puid PuidType) {
	s.put <- puid
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) Remove(puid PuidType) {
	s.del <- puid
}

// APIHANDLER
func tmbchkAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Tmbs []*TmbProp `json:"tmbs"`
	}

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if len(arg.Tmbs) == 0 {
		WriteError400(w, ErrNoData, AECtmbchknodata)
		return
	}

	for _, tp := range arg.Tmbs {
		if syspath, ok := syspathcache.Path(tp.PUID()); ok {
			if prop, err := propcache.Get(syspath); err == nil {
				tp.SetTmb(prop.(Pather).NTmb(), prop.(Pather).MTmb())
			}
		}
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID   IdType     `json:"aid"`
		PUIDs []PuidType `json:"puids"`
	}

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if len(arg.PUIDs) == 0 {
		WriteError400(w, ErrNoData, AECtmbscnnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECtmbscnnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	go func() {
		for _, puid := range arg.PUIDs {
			if syspath, ok := syspathcache.Path(puid); ok {
				if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
					ThumbScanner.Add(puid)
				}
			}
		}
	}()

	WriteOK(w, nil)
}

// The End.
