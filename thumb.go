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

type Mime_t int

const (
	MimeNil  Mime_t = 0
	MimeGif  Mime_t = 1
	MimePng  Mime_t = 2
	MimeJpeg Mime_t = 3
	MimeWebp Mime_t = 4
)

var MimeStr = map[Mime_t]string{
	MimeNil:  "",
	MimeGif:  "image/gif",
	MimePng:  "image/png",
	MimeJpeg: "image/jpeg",
	MimeWebp: "image/webp",
}

var MimeVal = map[string]Mime_t{
	"":           MimeNil,
	"image/gif":  MimeGif,
	"image/png":  MimePng,
	"image/jpeg": MimeJpeg,
	"image/webp": MimeWebp,
}

// MediaData is thumbnails cache element.
type MediaData struct {
	Data []byte
	Mime Mime_t
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
			tp.SetTmb(TMBreject, MimeNil)
			return
		}
		var md, ok = v.(*MediaData)
		if !ok {
			tp.SetTmb(TMBreject, MimeNil)
			return
		}
		tp.SetTmb(TMBcached, md.Mime)
	} else {
		tp.SetTmb(TMBnone, MimeNil)
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
func (tp *TmbProp) SetTmb(tmb int, mime Mime_t) {
	tp.NTmbVal = tmb
	tp.MTmbVal = MimeStr[mime]
}

// FindTmb finds thumbnail in embedded file tags, or build it if it possible.
func FindTmb(prop Pather, syspath string) (md *MediaData, err error) {
	if prop.Type() < 0 {
		err = ErrNotFile
		return // not a file
	}

	// try to extract from EXIF
	var orientation = OrientNormal
	if ek, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		if cfg.UseEmbeddedTmb {
			if md, err = GetExifTmb(syspath); err == nil {
				return // thumbnail from EXIF
			}
		}
		orientation = ek.Orientation
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

	return MakeTmb(file, orientation)
}

// MakeTmb reads image from the stream and makes thumbnail with format
// depended from alpha-channel is present in the original image.
func MakeTmb(r io.Reader, orientation int) (md *MediaData, err error) {
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

	return ToNativeImg(dst, ftype) // set valid thumbnail
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

	var queue []PuidType
	var ctx chan struct{}

	var cache = func(puid PuidType) {
		ctx = make(chan struct{})
		go func() {
			defer close(ctx)
			if puid != 0 {
				thumbcache.Get(puid)
			}
		}()
	}

	for {
		select {
		case puid1 := <-s.put:
			var found = false
			for _, puid2 := range queue {
				if puid1 == puid2 {
					found = true
					break
				}
			}
			if !found {
				if ctx == nil {
					cache(puid1)
				} else {
					queue = append(queue, puid1)
				}
			}
		case puid1 := <-s.del:
			for i, puid2 := range queue {
				if puid1 == puid2 {
					queue = append(queue[:i], queue[i+1:]...)
					break
				}
			}
		case <-ctx:
			if len(queue) > 0 {
				var puid = queue[0]
				queue = queue[1:]
				cache(puid)
			} else {
				ctx = nil
			}
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
				tp.SetTmb(prop.(Pather).NTmb(), MimeVal[prop.(Pather).MTmb()])
			}
		}
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnstartAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  IdType     `json:"aid"`
		List []PuidType `json:"list"`
	}

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, ErrNoData, AECscnstartnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECscnstartnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	for _, puid := range arg.List {
		if syspath, ok := syspathcache.Path(puid); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				ThumbScanner.Add(puid)
			}
		}
	}

	WriteOK(w, nil)
}

// APIHANDLER
func tmbscnbreakAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  IdType     `json:"aid"`
		List []PuidType `json:"list"`
	}

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, ErrNoData, AECscnbreaknodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, ErrNoAcc, AECscnbreaknoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	for _, puid := range arg.List {
		if syspath, ok := syspathcache.Path(puid); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				ThumbScanner.Remove(puid)
			}
		}
	}

	WriteOK(w, nil)
}

// The End.
