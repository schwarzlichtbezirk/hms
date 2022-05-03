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
	"strings"

	"github.com/disintegration/gift"
	_ "github.com/oov/psd" // register PSD format
	"github.com/schwarzlichtbezirk/wpk"
	_ "github.com/spate/glimage/dds" // register DDS format
	_ "golang.org/x/image/bmp"       // register BMP format
	_ "golang.org/x/image/tiff"      // register TIFF format
	_ "golang.org/x/image/webp"      // register WebP format

	_ "github.com/ftrvxmtrx/tga" // put TGA to end, decoder does not register magic prefix
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

type Mime_t int16

const (
	MimeDis  Mime_t = -1 // file can not be cached for thumbnails.
	MimeNil  Mime_t = 0  // file is not cached for thumbnails, have indeterminate state.
	MimeUnk  Mime_t = 1
	MimeGif  Mime_t = 2
	MimePng  Mime_t = 3
	MimeJpeg Mime_t = 4
	MimeWebp Mime_t = 5
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

func GetMimeVal(str string) Mime_t {
	if mime, ok := MimeVal[str]; ok {
		return mime
	}
	return MimeUnk
}

// MediaData is thumbnails cache element.
type MediaData struct {
	Data []byte
	Mime Mime_t
}

// TmbProp is thumbnails properties.
type TmbProp struct {
	PUIDVal Puid_t `json:"puid" yaml:"puid"`
	MTmbVal Mime_t `json:"mtmb" yaml:"mtmb"`
}

// Setup generates PUID (path unique identifier) and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.PUIDVal = syspathcache.Cache(syspath)
	if ts, ok := thumbpkg.Tagset(syspath); ok {
		if str, ok := ts.String(wpk.TIDmime); ok {
			if strings.HasPrefix(str, "image/") {
				tp.SetTmb(GetMimeVal(str))
			} else {
				tp.SetTmb(MimeDis)
			}
		} else {
			tp.SetTmb(MimeUnk)
		}
	} else {
		tp.SetTmb(MimeNil)
	}
}

// PUID returns thumbnail key, it's full system path unique ID.
func (tp *TmbProp) PUID() Puid_t {
	return tp.PUIDVal
}

// MTmb returns thumbnail MIME type, if thumbnail is present.
func (tp *TmbProp) MTmb() Mime_t {
	return tp.MTmbVal
}

// SetTmb updates thumbnail state to given value.
func (tp *TmbProp) SetTmb(mime Mime_t) {
	tp.MTmbVal = mime
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
		if cfg.UseEmbeddedTmb && ek.ThumbJpegLen > 0 {
			md = &MediaData{
				Data: ek.thumbJpeg,
				Mime: MimeJpeg,
			}
			return // thumbnail from EXIF
		}
		orientation = ek.Orientation
	}

	// try to extract from ID3
	if _, ok := prop.(*TagKit); ok { // skip non-ID3 properties
		if md, err = GetTagTmb(syspath); err == nil {
			return
		}
	}

	// check all others are images
	if GetFileGroup(prop.Name()) != FGimage {
		err = ErrNotImg
		return // file is not image
	}

	if prop.Size() > cfg.ThumbFileMaxSize {
		err = ErrTooBig
		return // file is too big
	}

	if md, err = GetCachedThumb(syspath, orientation); err != nil {
		return
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

// GetCachedThumb tries to extract existing thumbnail from cache, otherwise
// makes new one and put it to cache.
func GetCachedThumb(syspath string, orientation int) (md *MediaData, err error) {
	// try to extract thumbnail from package
	if md, err = thumbpkg.GetImage(syspath); err != nil || md != nil {
		return
	}

	var r io.ReadCloser
	if r, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer r.Close()

	if md, err = MakeThumb(r, orientation); err != nil {
		return
	}

	// push thumbnail to package
	err = thumbpkg.PutImage(syspath, md)
	return
}

// GetCachedEmbThumb tries to extract existing thumbnail from cache, otherwise
// reads image from the stream, makes new thumbnail and put it to cache.
func GetCachedEmbThumb(r io.Reader, fkey string) (md *MediaData, err error) {
	// try to extract thumbnail from package
	if md, err = thumbpkg.GetImage(fkey); err != nil || md != nil {
		return
	}

	if md, err = MakeThumb(r, OrientNormal); err != nil {
		return
	}

	// push thumbnail to package
	err = thumbpkg.PutImage(fkey, md)
	return
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
	put chan Puid_t
	del chan Puid_t
}

// Scan is goroutine for thumbnails scanning.
func (s *scanner) Scan() {
	s.put = make(chan Puid_t)
	s.del = make(chan Puid_t)

	var queue []Puid_t
	var ctx chan struct{}

	var cache = func(puid Puid_t) {
		ctx = make(chan struct{})
		go func() {
			defer close(ctx)
			if puid != 0 {
				var syspath, ok = syspathcache.Path(puid)
				if !ok {
					return // file path not found
				}

				var err error
				var prop interface{}
				if prop, err = propcache.Get(syspath); err != nil {
					return // can not get properties
				}
				var fp = prop.(Pather)
				if fp.MTmb() != MimeNil {
					return // thumbnail already scanned
				}
				var md *MediaData
				if md, err = FindTmb(fp, syspath); err != nil {
					fp.SetTmb(MimeDis)
					return
				}
				fp.SetTmb(md.Mime)
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
func (s *scanner) Add(puid Puid_t) {
	s.put <- puid
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) Remove(puid Puid_t) {
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
				tp.SetTmb(prop.(Pather).MTmb())
			}
		}
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnstartAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID  ID_t     `json:"aid"`
		List []Puid_t `json:"list"`
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
		AID  ID_t     `json:"aid"`
		List []Puid_t `json:"list"`
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
