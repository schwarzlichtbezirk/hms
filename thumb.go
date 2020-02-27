package hms

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"os"

	"github.com/bluele/gcache"
	"github.com/disintegration/gift"
)

const thumbside = 256

var thumbrect = image.Rect(0, 0, thumbside, thumbside)

var thumbcache = gcache.New(16 * 1024).LRU().Build()

var thumbfilter = gift.New(
	gift.Resize(thumbside, 0, gift.NearestNeighborResampling),
	gift.CropToSize(thumbside, thumbside, gift.CenterAnchor),
)

var thumbpngenc = png.Encoder{
	CompressionLevel: png.BestCompression,
}

func ThumbName(fname string) string {
	var h = md5.Sum([]byte(fname))
	return hex.EncodeToString(h[:])
}

func ThumbImg(fname string) (img image.Image, err error) {
	var file *os.File
	file, err = os.Open(fname)
	if err != nil {
		return
	}
	defer file.Close()

	var ft string
	var src image.Image
	src, ft, err = image.Decode(file)
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
	Log.Println(fname, ft, src.Bounds().Dx(), src.Bounds().Dy())
	return
}

func CacheImg(fp *FileProp) (key string, tmb []byte) {
	var err error
	key = ThumbName(fp.Path)

	var val interface{}
	if val, err = thumbcache.Get(key); err != nil {
		if val != nil {
			tmb = val.([]byte)
		}
		return // image already cached
	}

	var img image.Image
	defer thumbcache.Set(key, tmb)

	if img, err = ThumbImg(fp.Path); err != nil {
		return // can not make thumbnail
	}

	var buf bytes.Buffer
	if err = thumbpngenc.Encode(&buf, img); err != nil {
		return // can not write png
	}
	tmb = buf.Bytes()
	/*{
		var f, err = os.Create("d:/temp/"+key+".png")
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
	var route = r.URL.Path
	var val, err = thumbcache.Get(route)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var content, ok = val.([]byte)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, route, starttime, bytes.NewReader(content))
}

// APIHANDLER
func tmbgetApi(w http.ResponseWriter, r *http.Request) {
}

// The End.
