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
	ErrNoThumb  = errors.New("music file without thumbnail")
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

// Puider helps to get PUID from some properties kit.
type Puider interface {
	PUID() Puid_t // path unique ID encoded to hex-base32
}

// PuidProp encapsulated path unique ID value for some properties kit.
type PuidProp struct {
	PUIDVal Puid_t `json:"puid" yaml:"puid" xml:"puid"`
}

func (pp *PuidProp) Setup(syspath string) {
	pp.PUIDVal = syspathcache.Cache(syspath)
}

// PUID returns thumbnail key, it's full system path unique ID.
func (pp *PuidProp) PUID() Puid_t {
	return pp.PUIDVal
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
	ETmbVal Mime_t `json:"etmb" yaml:"etmb" xml:"etmb"`
	MTmbVal Mime_t `json:"mtmb" yaml:"mtmb" xml:"mtmb"`
	MT02Val Mime_t `json:"mt02,omitempty" yaml:"mt02,omitempty" xml:"mt02,omitempty"`
	MT03Val Mime_t `json:"mt03,omitempty" yaml:"mt03,omitempty" xml:"mt03,omitempty"`
	MT04Val Mime_t `json:"mt04,omitempty" yaml:"mt04,omitempty" xml:"mt04,omitempty"`
	MT06Val Mime_t `json:"mt06,omitempty" yaml:"mt06,omitempty" xml:"mt06,omitempty"`
	MT08Val Mime_t `json:"mt08,omitempty" yaml:"mt08,omitempty" xml:"mt08,omitempty"`
	MT09Val Mime_t `json:"mt09,omitempty" yaml:"mt09,omitempty" xml:"mt09,omitempty"`
	MT10Val Mime_t `json:"mt10,omitempty" yaml:"mt10,omitempty" xml:"mt10,omitempty"`
	MT12Val Mime_t `json:"mt12,omitempty" yaml:"mt12,omitempty" xml:"mt12,omitempty"`
	MT15Val Mime_t `json:"mt15,omitempty" yaml:"mt15,omitempty" xml:"mt15,omitempty"`
	MT16Val Mime_t `json:"mt16,omitempty" yaml:"mt16,omitempty" xml:"mt16,omitempty"`
	MT18Val Mime_t `json:"mt18,omitempty" yaml:"mt18,omitempty" xml:"mt18,omitempty"`
	MT20Val Mime_t `json:"mt20,omitempty" yaml:"mt20,omitempty" xml:"mt20,omitempty"`
	MT24Val Mime_t `json:"mt24,omitempty" yaml:"mt24,omitempty" xml:"mt24,omitempty"`
	MT30Val Mime_t `json:"mt30,omitempty" yaml:"mt30,omitempty" xml:"mt30,omitempty"`
	MT36Val Mime_t `json:"mt36,omitempty" yaml:"mt36,omitempty" xml:"mt36,omitempty"`
}

const (
	htcell = 24 // horizontal tile cell length
	vtcell = 18 // vertical tile cell length
)

type TM_t int

const (
	tme  TM_t = -1
	tm0  TM_t = 0
	tm2  TM_t = 2
	tm3  TM_t = 3
	tm4  TM_t = 4
	tm6  TM_t = 6
	tm8  TM_t = 8
	tm9  TM_t = 9
	tm10 TM_t = 10
	tm12 TM_t = 12
	tm15 TM_t = 15
	tm16 TM_t = 16
	tm18 TM_t = 18
	tm20 TM_t = 20
	tm24 TM_t = 24
	tm30 TM_t = 30
	tm36 TM_t = 36
)

var tmset = [...]TM_t{2, 3, 4, 6, 8, 9, 10, 12, 15, 16, 18, 20, 24, 30, 36}

var tmbdis = TmbProp{
	ETmbVal: MimeDis,
	MTmbVal: MimeDis,
	MT02Val: MimeDis,
	MT03Val: MimeDis,
	MT04Val: MimeDis,
	MT06Val: MimeDis,
	MT08Val: MimeDis,
	MT10Val: MimeDis,
	MT12Val: MimeDis,
	MT15Val: MimeDis,
	MT16Val: MimeDis,
	MT18Val: MimeDis,
	MT20Val: MimeDis,
	MT24Val: MimeDis,
	MT30Val: MimeDis,
	MT36Val: MimeDis,
}

// Thumber helps to cast some properties kit to TmbProp struct.
type Thumber interface {
	Tmb() *TmbProp            // returns self pointers for embedded structures
	Tile(TM_t) (Mime_t, bool) // tile MIME type, -1 - can not make thumbnail; 0 - not cached; >=1 - cached
	SetTile(TM_t, Mime_t) bool
}

// Tmb is Thumber interface implementation.
func (tp *TmbProp) Tmb() *TmbProp {
	return tp
}

// Setup generates PUID (path unique identifier) and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.ETmbVal = MimeDis // setup as default
	if ts, ok := thumbpkg.Tagset(syspath); ok {
		if str, ok := ts.TagStr(wpk.TIDmime); ok {
			if strings.HasPrefix(str, "image/") {
				tp.MTmbVal = GetMimeVal(str)
			} else {
				tp.MTmbVal = MimeDis
			}
		} else {
			tp.MTmbVal = MimeUnk
		}
	} else {
		tp.MTmbVal = MimeNil
	}
	for _, tm := range tmset {
		var tilepath = fmt.Sprintf("%s?%dx%d", syspath, tm*htcell, tm*vtcell)
		if ts, ok := tilespkg.Tagset(tilepath); ok {
			if str, ok := ts.TagStr(wpk.TIDmime); ok {
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

// Tile returns image MIME type with given tile multiplier.
func (tp *TmbProp) Tile(tm TM_t) (mime Mime_t, ok bool) {
	ok = true
	switch tm {
	case tme:
		mime = tp.ETmbVal
	case tm0:
		mime = tp.MTmbVal
	case tm2:
		mime = tp.MT02Val
	case tm3:
		mime = tp.MT03Val
	case tm4:
		mime = tp.MT04Val
	case tm6:
		mime = tp.MT06Val
	case tm8:
		mime = tp.MT08Val
	case tm9:
		mime = tp.MT09Val
	case tm10:
		mime = tp.MT10Val
	case tm12:
		mime = tp.MT12Val
	case tm15:
		mime = tp.MT15Val
	case tm16:
		mime = tp.MT16Val
	case tm18:
		mime = tp.MT18Val
	case tm20:
		mime = tp.MT20Val
	case tm24:
		mime = tp.MT24Val
	case tm30:
		mime = tp.MT30Val
	case tm36:
		mime = tp.MT36Val
	default:
		ok = false
	}
	return
}

// SetTile updates image state to given value for tile with given tile multiplier.
func (tp *TmbProp) SetTile(tm TM_t, mime Mime_t) (ok bool) {
	ok = true
	switch tm {
	case tme:
		tp.ETmbVal = mime
	case tm0:
		tp.MTmbVal = mime
	case tm2:
		tp.MT02Val = mime
	case tm3:
		tp.MT03Val = mime
	case tm4:
		tp.MT04Val = mime
	case tm6:
		tp.MT06Val = mime
	case tm8:
		tp.MT08Val = mime
	case tm9:
		tp.MT09Val = mime
	case tm10:
		tp.MT10Val = mime
	case tm12:
		tp.MT12Val = mime
	case tm15:
		tp.MT15Val = mime
	case tm16:
		tp.MT16Val = mime
	case tm18:
		tp.MT18Val = mime
	case tm20:
		tp.MT20Val = mime
	case tm24:
		tp.MT24Val = mime
	case tm30:
		tp.MT30Val = mime
	case tm36:
		tp.MT36Val = mime
	default:
		ok = false
	}
	return
}

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
	if GetFileGroup(syspath) != FGimage {
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
	if GetFileGroup(syspath) != FGimage {
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
	if tmb, ok := prop.(Thumber); ok {
		var tp = tmb.Tmb()
		if tp.MTmbVal != MimeNil {
			return // thumbnail already scanned
		}

		var md *MediaData
		if md, err = GetCachedThumb(string(fpath)); err != nil {
			tp.MTmbVal = MimeDis
			return
		}
		tp.MTmbVal = md.Mime
	}
}

// TilePath is tile path type for cache processing.
type TilePath struct {
	Path string
	Wdh  int
	Hgt  int
}

// Cache is Cacher implementation for TilePath type.
func (tile TilePath) Cache() {
	var err error
	var prop interface{}
	if prop, err = propcache.Get(tile.Path); err != nil {
		return // can not get properties
	}
	if tmb, ok := prop.(Thumber); ok {
		var tp = tmb.Tmb()
		var tm = TM_t(tile.Wdh / htcell)
		if mime, ok := tp.Tile(tm); ok && mime != MimeNil {
			return // thumbnail already scanned
		}

		var md *MediaData
		if md, err = GetCachedTile(tile.Path, tile.Wdh, tile.Hgt); err != nil {
			tp.SetTile(tm, MimeDis)
			return
		}
		tp.SetTile(tm, md.Mime)
	}
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
func (s *scanner) AddTile(syspath string, tm TM_t) {
	s.put <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) RemoveTile(syspath string, tm TM_t) {
	s.del <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
}

// APIHANDLER
func tilechkAPI(w http.ResponseWriter, r *http.Request) {
	type TmbKit struct {
		PuidProp `yaml:",inline"`
		TmbProp  `yaml:",inline"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		List []Puid_t `json:"list" yaml:"list" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Tmbs []TmbKit `json:"tmbs" yaml:"tmbs" xml:"list>tmb"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtilechknodata)
		return
	}

	for _, puid := range arg.List {
		if syspath, ok := syspathcache.Path(puid); ok {
			var tk TmbKit
			tk.PUIDVal = puid
			if prop, err := propcache.Get(syspath); err == nil {
				if tmb, ok := prop.(Thumber); ok {
					var tp = tmb.Tmb()
					tk.TmbProp = *tp
				} else {
					tk.TmbProp = tmbdis
				}
			} else {
				tk.TmbProp = tmbdis
			}
			ret.Tmbs = append(ret.Tmbs, tk)
		}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func tilescnstartAPI(w http.ResponseWriter, r *http.Request) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t     `json:"aid" yaml:"aid" xml:"aid,attr"`
		List []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
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

	for _, ttm := range arg.List {
		if syspath, ok := syspathcache.Path(ttm.PUID); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				if ttm.TM == tm0 {
					ImgScanner.AddTmb(syspath)
				} else {
					ImgScanner.AddTile(syspath, ttm.TM)
				}
			}
		}
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func tilescnbreakAPI(w http.ResponseWriter, r *http.Request) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		AID  ID_t     `json:"aid" yaml:"aid" xml:"aid,attr"`
		List []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
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

	for _, ttm := range arg.List {
		if syspath, ok := syspathcache.Path(ttm.PUID); ok {
			if cg := prf.PathAccess(syspath, auth == prf); !cg.IsZero() {
				if ttm.TM == tm0 {
					ImgScanner.RemoveTmb(syspath)
				} else {
					ImgScanner.RemoveTile(syspath, ttm.TM)
				}
			}
		}
	}

	WriteOK(w, r, nil)
}

// The End.
