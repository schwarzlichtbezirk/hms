package hms

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bluele/gcache"
	"github.com/disintegration/gift"
)

const (
	TMB_none   = 0
	TMB_reject = -1
	TMB_cached = 1
)

const thumbside = 256

var thumbrect = image.Rect(0, 0, thumbside, thumbside)

var thumbcache = gcache.New(50000).LRU().Build()

var thumbfilter = gift.New(
	gift.Resize(thumbside, 0, gift.NearestNeighborResampling),
	gift.CropToSize(thumbside, thumbside, gift.CenterAnchor),
)

var thumbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

// Data for "entchk" API handler.
type tmbchkArg struct {
	ITmbs []FileProp `json:"itmbs"`
}

// Argument data for "tmbscn" API handler.
type tmbscnArg struct {
	ITmbs []FileProp `json:"itmbs"`
	Force bool       `json:"force"`
}

func ThumbName(fname string) string {
	var h = md5.Sum([]byte(fname))
	return hex.EncodeToString(h[:])
}

func ThumbImg(fname string) (img image.Image, ftype string, err error) {
	var file *os.File
	file, err = os.Open(fname)
	if err != nil {
		return
	}
	defer file.Close()

	var src image.Image
	src, ftype, err = image.Decode(file)
	if err != nil {
		return
	}
	if src.Bounds().In(thumbrect) {
		img = src
		return
	}
	var dst = image.NewRGBA(thumbfilter.Bounds(src.Bounds()))
	thumbfilter.Draw(dst, src)
	img = dst
	return
}

func CacheImg(fp *FileProp, force bool) (ftmb []byte) {
	var err error

	if !force {
		var val interface{}
		if val, err = thumbcache.Get(fp.KTmb); err == nil {
			if val != nil {
				ftmb = val.([]byte)
			}
			return // image already cached
		}
	}

	defer func() {
		if len(ftmb) > 0 {
			fp.NTmb = TMB_cached
			thumbcache.Set(fp.KTmb, ftmb)
		} else {
			fp.NTmb = TMB_reject
			thumbcache.Set(fp.KTmb, nil)
		}
	}()

	var img image.Image
	if img, _, err = ThumbImg(fp.Path); err != nil {
		return // can not make thumbnail
	}

	var buf bytes.Buffer
	if err = thumbpngenc.Encode(&buf, img); err != nil {
		return // can not write png
	}
	ftmb = buf.Bytes()
	/*{
		var f, err = os.Create("d:/temp/"+fp.KTmb+".png")
		if err != nil {
			Log.Println(err.Error())
			return
		}
		defer f.Close()
		f.Write(buf.Bytes())
	}*/
	return // set valid thumbnail
}

// Hands out thumbnails for given files if them cached.
func thumbHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var ktmb = r.URL.Path[len(r.URL.Path)-32:]
	var val interface{}
	if val, err = thumbcache.Get(ktmb); err != nil {
		http.NotFound(w, r)
		return
	}
	var content, ok = val.([]byte)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, ktmb+".png", starttime, bytes.NewReader(content))
}

// APIHANDLER
func tmbchkApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg tmbchkArg

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_tmbchkbadreq)
			return
		}
		if len(arg.ITmbs) == 0 {
			WriteError400(w, ErrNoData, EC_tmbchknodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_tmbchknoreq)
		return
	}

	for i := range arg.ITmbs {
		var fp = &arg.ITmbs[i]
		if tmb, err := thumbcache.Get(fp.KTmb); err == nil {
			if tmb != nil {
				fp.NTmb = TMB_cached
			} else {
				fp.NTmb = TMB_reject
			}
		} else {
			fp.NTmb = TMB_none
		}
	}

	WriteJson(w, http.StatusOK, arg)
}

// APIHANDLER
func tmbscnApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg tmbscnArg

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_tmbscnbadreq)
			return
		}
		if len(arg.ITmbs) == 0 {
			WriteError400(w, ErrNoData, EC_tmbscnnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_tmbscnnoreq)
		return
	}

	for i := range arg.ITmbs {
		var fp = &arg.ITmbs[i]
		CacheImg(fp, arg.Force)
	}

	WriteJson(w, http.StatusOK, nil)
}

// The End.
