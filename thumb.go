package hms

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/bluele/gcache"
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

const thumbside = 256

var thumbrect = image.Rect(0, 0, thumbside, thumbside)

// Thumbnails cache.
// Key - thumbnail key (syspath MD5-hash), value - thumbnail image.
var thumbcache = gcache.New(0).
	Simple().
	LoaderFunc(func(key interface{}) (ret interface{}, err error) {
		var syspath, ok = ktmbcache.Path(key.(string))
		if !ok {
			err = ErrNoHash
			return // file path not found
		}

		var cp interface{}
		if cp, err = propcache.Get(syspath); err != nil {
			return // can not get properties
		}
		var prop = cp.(Proper)
		if prop.NTmb() == TMB_reject {
			err = ErrNotThumb
			return // thumbnail rejected
		}

		var tmb *ThumbElem
		if tmb, err = FindTmb(prop, syspath); tmb != nil {
			prop.SetNTmb(TMB_cached)
			ret = tmb
		} else {
			prop.SetNTmb(TMB_reject)
		}
		return // ok
	}).
	Build()

var thumbfilter = gift.New(
	gift.ResizeToFit(thumbside, thumbside, gift.LinearResampling),
)

var thumbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// Cache with hash/syspath and syspath/hash values.
type KeyThumbCache struct {
	keypath map[string]string
	pathkey map[string]string
	mux     sync.RWMutex
}

func (c *KeyThumbCache) Key(syspath string) (ktmb string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	ktmb, ok = c.pathkey[syspath]
	return
}

func (c *KeyThumbCache) Path(ktmb string) (syspath string, ok bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	syspath, ok = c.keypath[ktmb]
	return
}

func (c *KeyThumbCache) Cache(syspath string) string {
	if ktmb, ok := c.Key(syspath); ok {
		return ktmb
	}
	var h = md5.Sum([]byte(syspath))
	var ktmb = hex.EncodeToString(h[:])
	c.mux.Lock()
	defer c.mux.Unlock()
	c.pathkey[syspath] = ktmb
	c.keypath[ktmb] = syspath
	return ktmb
}

var ktmbcache = KeyThumbCache{
	keypath: map[string]string{},
	pathkey: map[string]string{},
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
	KTmbVal string `json:"ktmb,omitempty"`
	NTmbVal int    `json:"ntmb,omitempty"`
}

// Generates cache key as hash of path and updates cached state.
func (tp *TmbProp) Setup(syspath string) {
	tp.KTmbVal = ktmbcache.Cache(syspath)
	tp.NTmbCached()
}

// Updates cached state for this cache key.
func (tp *TmbProp) NTmbCached() {
	if thumbcache.Has(tp.KTmbVal) {
		tp.NTmbVal = TMB_cached
	} else {
		tp.NTmbVal = TMB_none
	}
}

// Thumbnail key, it's MD5-hash of full path.
func (tp *TmbProp) KTmb() string {
	return tp.KTmbVal
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

	if prop.Size() > cfg.ThumbMaxFile {
		err = ErrTooBig
		return // file is too big
	}

	// check all others are images
	if typetogroup[prop.Type()] != FG_image {
		err = ErrNotImg
		return // file is not image
	}

	return MakeTmb(syspath)
}

func MakeTmb(syspath string) (tmb *ThumbElem, err error) {
	var file *os.File
	if file, err = os.Open(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var ftype string
	var src, dst image.Image
	if src, ftype, err = image.Decode(file); err != nil {
		return // can not decode file by any codec
	}
	if src.Bounds().In(thumbrect) {
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

// Hands out thumbnails for given files if them cached.
func thumbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var ktmb = r.URL.Path[len(r.URL.Path)-32:]
	var val interface{}
	if val, err = thumbcache.Get(ktmb); err != nil {
		WriteError(w, http.StatusNotFound, err, EC_thumbabsent)
		return
	}
	var tmb, ok = val.(*ThumbElem)
	if !ok {
		WriteError500(w, ErrBadThumb, EC_thumbbadcnt)
		return
	}
	if tmb == nil {
		WriteError(w, http.StatusNotFound, ErrNotThumb, EC_thumbnotcnt)
		return
	}
	w.Header().Set("Content-Type", tmb.Mime)
	http.ServeContent(w, r, ktmb, starttime, bytes.NewReader(tmb.Data))
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
		if syspath, ok := ktmbcache.Path(tp.KTmb()); ok {
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
	if acc = AccList.ByID(int(arg.AID)); acc == nil {
		WriteError400(w, ErrNoAcc, EC_tmbscnnoacc)
		return
	}

	go func() {
		for _, shrpath := range arg.Paths {
			var syspath = acc.GetSharePath(shrpath)
			if ktmb, ok := ktmbcache.Key(syspath); ok {
				thumbcache.Get(ktmb)
			}
		}
	}()

	WriteOK(w, nil)
}

// The End.
