package hms

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"

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

// Tiles multipliers:
//  576px: 2,  4,  6,  8, 10, 12
//  768px: 3,  6,  9, 12, 15, 18
// 1280px: 4,  8, 12, 16, 20, 24
// 1920px: 6, 12, 18, 24, 30, 36

// Tiles horizontal resolutions (tm x 24):
//  576px:  48,  96, 144, 192, 240, 288
//  768px:  72, 144, 216, 288, 360, 432
// 1280px:  96, 192, 288, 384, 480, 576
// 1920px: 144, 288, 432, 576, 720, 864

// TmbProp is thumbnails properties.
type TmbProp struct {
	PUIDVal Puid_t `json:"puid" yaml:"puid" xml:"puid"`
	MTmbVal Mime_t `json:"mtmb" yaml:"mtmb" xml:"mtmb"`
	TM02Val Mime_t `json:"tm02,omitempty" yaml:"tm02,omitempty" xml:"tm02,omitempty"`
	TM03Val Mime_t `json:"tm03,omitempty" yaml:"tm03,omitempty" xml:"tm03,omitempty"`
	TM04Val Mime_t `json:"tm04,omitempty" yaml:"tm04,omitempty" xml:"tm04,omitempty"`
	TM06Val Mime_t `json:"tm06,omitempty" yaml:"tm06,omitempty" xml:"tm06,omitempty"`
	TM08Val Mime_t `json:"tm08,omitempty" yaml:"tm08,omitempty" xml:"tm08,omitempty"`
	TM09Val Mime_t `json:"tm09,omitempty" yaml:"tm09,omitempty" xml:"tm09,omitempty"`
	TM10Val Mime_t `json:"tm10,omitempty" yaml:"tm10,omitempty" xml:"tm10,omitempty"`
	TM12Val Mime_t `json:"tm12,omitempty" yaml:"tm12,omitempty" xml:"tm12,omitempty"`
	TM15Val Mime_t `json:"tm15,omitempty" yaml:"tm15,omitempty" xml:"tm15,omitempty"`
	TM16Val Mime_t `json:"tm16,omitempty" yaml:"tm16,omitempty" xml:"tm16,omitempty"`
	TM18Val Mime_t `json:"tm18,omitempty" yaml:"tm18,omitempty" xml:"tm18,omitempty"`
	TM20Val Mime_t `json:"tm20,omitempty" yaml:"tm20,omitempty" xml:"tm20,omitempty"`
	TM24Val Mime_t `json:"tm24,omitempty" yaml:"tm24,omitempty" xml:"tm24,omitempty"`
	TM30Val Mime_t `json:"tm30,omitempty" yaml:"tm30,omitempty" xml:"tm30,omitempty"`
	TM36Val Mime_t `json:"tm36,omitempty" yaml:"tm36,omitempty" xml:"tm36,omitempty"`
}

var tmset = [...]int{2, 3, 4, 6, 8, 9, 10, 12, 15, 16, 18, 20, 24, 30, 36}

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
	for _, tm := range tmset {
		var tilepath = fmt.Sprintf("%s?%dx%d", syspath, tm*24, tm*18)
		if ts, ok := tilespkg.Tagset(tilepath); ok {
			if str, ok := ts.String(wpk.TIDmime); ok {
				if strings.HasPrefix(str, "image/") {
					tp.SetTile(tm, GetMimeVal(str))
				} else {
					tp.SetTile(tm, MimeDis)
				}
			} else {
				tp.SetTile(tm, MimeUnk)
			}
		} else {
			tp.SetTile(tm, MimeNil)
		}
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

// Tile returns image MIME type with given tile multiplier.
func (tp *TmbProp) Tile(tm int) (mime Mime_t, ok bool) {
	ok = true
	switch tm {
	case 2:
		mime = tp.TM02Val
	case 3:
		mime = tp.TM03Val
	case 4:
		mime = tp.TM04Val
	case 6:
		mime = tp.TM06Val
	case 8:
		mime = tp.TM08Val
	case 9:
		mime = tp.TM09Val
	case 10:
		mime = tp.TM10Val
	case 12:
		mime = tp.TM12Val
	case 15:
		mime = tp.TM15Val
	case 16:
		mime = tp.TM16Val
	case 18:
		mime = tp.TM18Val
	case 20:
		mime = tp.TM20Val
	case 24:
		mime = tp.TM24Val
	case 30:
		mime = tp.TM30Val
	case 36:
		mime = tp.TM36Val
	default:
		ok = false
	}
	return
}

// SetTile updates image state to given value for tile with given tile multiplier.
func (tp *TmbProp) SetTile(tm int, mime Mime_t) (ok bool) {
	ok = true
	switch tm {
	case 2:
		tp.TM02Val = mime
	case 3:
		tp.TM03Val = mime
	case 4:
		tp.TM04Val = mime
	case 6:
		tp.TM06Val = mime
	case 8:
		tp.TM08Val = mime
	case 9:
		tp.TM09Val = mime
	case 10:
		tp.TM10Val = mime
	case 12:
		tp.TM12Val = mime
	case 15:
		tp.TM15Val = mime
	case 16:
		tp.TM16Val = mime
	case 18:
		tp.TM18Val = mime
	case 20:
		tp.TM20Val = mime
	case 24:
		tp.TM24Val = mime
	case 30:
		tp.TM30Val = mime
	case 36:
		tp.TM36Val = mime
	default:
		ok = false
	}
	return
}

// FindTmb finds thumbnail in embedded file tags, or build it if it possible.
func FindTmb(prop Pather, syspath string) (md *MediaData, err error) {
	// try to extract from EXIF
	var orientation = OrientNormal
	if ek, ok := prop.(*ExifKit); ok { // skip non-EXIF properties
		if cfg.UseEmbeddedTmb && ek.ThumbJpegLen > 0 {
			md = &ek.thumb
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

// Cacher provides function to perform image converting.
type Cacher interface {
	Cache()
}

// ThumbPath is thumbnail path type for cache processing.
type ThumbPath string

// Cache is Cacher implementation for ThumbPath type.
func (fpath ThumbPath) Cache() {
	var err error
	var prop interface{}
	if prop, err = propcache.Get(string(fpath)); err != nil {
		return // can not get properties
	}
	var fp = prop.(Pather)
	if fp.MTmb() != MimeNil {
		return // thumbnail already scanned
	}
	var md *MediaData
	if md, err = FindTmb(fp, string(fpath)); err != nil {
		fp.SetTmb(MimeDis)
		return
	}
	fp.SetTmb(md.Mime)
}

// TilePath is tile path type for cache processing.
type TilePath struct {
	Path string
	Wdh  int
	Hgt  int
}

// Cache is Cacher implementation for TilePath type.
func (tp TilePath) Cache() {
	var err error
	var prop interface{}
	if prop, err = propcache.Get(tp.Path); err != nil {
		return // can not get properties
	}
	var fp = prop.(Pather)
	var tm = tp.Wdh / 24
	if mime, ok := fp.Tile(tm); ok && mime != MimeNil {
		return // thumbnail already scanned
	}
	var md *MediaData
	if md, err = GetCachedTile(tp.Path, tp.Wdh, tp.Hgt); err != nil {
		fp.SetTile(tm, MimeDis)
		return
	}
	fp.SetTile(tm, md.Mime)
}

// ImgScanner is singleton for thumbnails producing
// with single queue to prevent overload.
var ImgScanner scanner

type scanner struct {
	put    chan Cacher
	del    chan Cacher
	cancel context.CancelFunc
	fin    context.Context
}

// Scan is goroutine for thumbnails scanning.
func (s *scanner) Scan() {
	s.put = make(chan Cacher)
	s.del = make(chan Cacher)
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())
	var cancel context.CancelFunc
	s.fin, cancel = context.WithCancel(context.Background())
	defer func() {
		s.fin, s.cancel = nil, nil
		cancel()
	}()

	var thrnum = cfg.ScanThreadsNum
	if thrnum == 0 {
		thrnum = runtime.GOMAXPROCS(0)
	}
	var busy = make([]bool, thrnum)
	var free = make(chan int)
	var args = make([]chan Cacher, thrnum)
	for i := range args {
		args[i] = make(chan Cacher)
	}

	var queue []Cacher

	var wg sync.WaitGroup
	wg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		var i = i // localize
		go func() {
			defer wg.Done()
			for {
				select {
				case arg := <-args[i]:
					arg.Cache()
					free <- i
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	func() {
		for {
			select {
			case arg := <-s.put:
				var found = false
				for i, b := range busy {
					if !b {
						busy[i] = true
						args[i] <- arg
						found = true
						break
					}
				}
				if !found {
					queue = append(queue, arg)
				}
			case arg := <-s.del:
				for i, val := range queue {
					if arg == val {
						queue = append(queue[:i], queue[i+1:]...)
						break
					}
				}
			case i := <-free:
				if len(queue) > 0 {
					var arg = queue[0]
					queue = queue[1:]
					busy[i] = true
					args[i] <- arg
				} else {
					busy[i] = false
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
}

// Stop makes the break to scanning process and returns context
// that indicates graceful scanning end.
func (s *scanner) Stop() (ctx context.Context) {
	ctx = s.fin
	if s.cancel != nil {
		s.cancel()
	}
	return
}

// AddTmb adds PUID to queue to make thumbnails.
func (s *scanner) AddTmb(syspath string) {
	s.put <- ThumbPath(syspath)
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) RemoveTmb(syspath string) {
	s.del <- ThumbPath(syspath)
}

// AddTile adds PUID to queue to make tile with given tile multiplier.
func (s *scanner) AddTile(syspath string, tm int) {
	s.put <- TilePath{syspath, tm * 24, tm * 18}
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) RemoveTile(syspath string, tm int) {
	s.del <- TilePath{syspath, tm * 24, tm * 18}
}

// APIHANDLER
func tmbchkAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Puids []Puid_t `json:"puids" yaml:"puids" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Tmbs []TmbProp `json:"tmbs" yaml:"tmbs" xml:"list>tmb"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Puids) == 0 {
		WriteError400(w, r, ErrNoData, AECtmbchknodata)
		return
	}

	for _, puid := range arg.Puids {
		if syspath, ok := syspathcache.Path(puid); ok {
			if prop, err := propcache.Get(syspath); err == nil {
				var tmb = TmbProp{
					PUIDVal: puid,
					MTmbVal: prop.(Pather).MTmb(),
				}
				ret.Tmbs = append(ret.Tmbs, tmb)
			}
		}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func tmbscnstartAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t     `json:"aid" yaml:"aid" xml:"aid,attr"`
		List []Puid_t `json:"list" yaml:"list" xml:"list>puid"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECscnstartnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECscnstartnoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	for _, puid := range arg.List {
		if syspath, ok := syspathcache.Path(puid); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				ImgScanner.AddTmb(syspath)
			}
		}
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func tmbscnbreakAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t     `json:"aid" yaml:"aid" xml:"aid,attr"`
		List []Puid_t `json:"list" yaml:"list" xml:"list>puid"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECscnbreaknodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByID(arg.AID); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECscnbreaknoacc)
		return
	}
	var auth *Profile
	if auth, err = GetAuth(w, r); err != nil {
		return
	}

	for _, puid := range arg.List {
		if syspath, ok := syspathcache.Path(puid); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				ImgScanner.RemoveTmb(syspath)
			}
		}
	}

	WriteOK(w, r, nil)
}

// The End.
