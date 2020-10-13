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

var thumbcache = gcache.New(0).Simple().Build()

var thumbfilter = gift.New(
	gift.ResizeToFit(thumbside, thumbside, gift.LinearResampling),
)

var thumbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// HTTP error messages
var (
	ErrBadThumb = errors.New("thumbnail content not cached")
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
	tp.KTmbVal = ThumbName(syspath)
	tp.UpdateTmb()
}

// Updates cached state for this cache key.
func (tp *TmbProp) UpdateTmb() {
	if tmb, err := thumbcache.Get(tp.KTmbVal); err == nil {
		if tmb != nil {
			tp.NTmbVal = TMB_cached
		} else {
			tp.NTmbVal = TMB_reject
		}
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

func ThumbName(syspath string) string {
	var h = md5.Sum([]byte(syspath))
	return hex.EncodeToString(h[:])
}

func CacheImg(fp Proper, syspath string, force bool) (tmb *ThumbElem) {
	var err error
	var ktmb = fp.KTmb()

	if !force {
		var val interface{}
		if val, err = thumbcache.Get(ktmb); err == nil {
			if val != nil {
				tmb = val.(*ThumbElem)
			}
			return // image already cached
		}
	}

	defer func() {
		if tmb != nil {
			fp.SetNTmb(TMB_cached)
			thumbcache.Set(ktmb, tmb)
		} else {
			fp.SetNTmb(TMB_reject)
			thumbcache.Set(ktmb, nil)
		}
	}()

	if typetogroup[fp.Type()] != FG_image {
		return // file is not image
	}

	if fp.Size() > cfg.ThumbMaxFile {
		return // file is too big
	}

	var file *os.File
	var ftype string
	var src, dst image.Image
	if file, err = os.Open(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

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
		WriteError(w, http.StatusNotFound, ErrBadThumb, EC_thumbbadcnt)
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
		tp.UpdateTmb()
	}

	WriteOK(w, arg)
}

// APIHANDLER
func tmbscnApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		AID   int      `json:"aid"`
		Paths []string `json:"paths"`
		Force bool     `json:"force"`
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
			if cp, err := propcache.Get(syspath); err == nil { // extract from cache
				var prop = cp.(Proper)
				CacheImg(prop, syspath, arg.Force)
			} else if fi, err := os.Stat(syspath); err == nil { // put into cache
				var prop = MakeProp(syspath, fi)
				CacheImg(prop, syspath, arg.Force)
			}
		}
	}()

	WriteOK(w, nil)
}

// The End.
