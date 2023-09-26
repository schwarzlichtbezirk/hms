package main

import (
	"fmt"
	"image"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/schwarzlichtbezirk/hms"
	. "github.com/schwarzlichtbezirk/hms/config"
	. "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"xorm.io/xorm"
)

type FileMap = map[string]struct{}

// FileList forms list of files to process by caching algorithm.
func FileList(fsys FS, list FileMap) (err error) {
	fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // file is directory
		}
		var ext = GetFileExt(fpath)
		if !IsTypeDecoded(ext) {
			return nil // file is not image
		}
		var fullpath = JoinFast(string(fsys), fpath)
		if !IsCached(fullpath) {
			list[fullpath] = struct{}{}
		}
		return nil
	})
	return
}

type ConvStat struct {
	filecount uint64
	tilecount uint64
	tmbcount  uint64
	filesize  uint64
	tilesize  uint64
	tmbsize   uint64
}

// Tiles multipliers:
var tilemult = [...]int{
	2, 3, 4, 6, 8, 9, 10, 12, 15, 16, 18, 20, 24, 30, 36,
}

// IsCached returns "true" if thumbnail and all tiles are cached for given filename.
func IsCached(fpath string) bool {
	if !ThumbPkg.HasTagset(fpath) {
		return false
	}

	for _, tm := range tilemult {
		var wdh, hgt = tm * 24, tm * 18
		var tilepath = fmt.Sprintf("%s?%dx%d", fpath, wdh, hgt)
		if !TilesPkg.HasTagset(tilepath) {
			return false
		}
	}

	return true
}

func CacheTags(fpath string, session *xorm.Session, cs *ConvStat) (err error) {
	var ext = GetFileExt(fpath)
	if !IsTypeEXIF(ext) {
		return
	}

	var puid = PathStoreCache(session, fpath)
	if _, ok := ExifStoreGet(session, puid); ok {
		return
	}

	var file File
	if file, err = OpenFile(fpath); err != nil {
		return
	}
	defer file.Close()

	if _, err = ExifExtract(session, file, puid); err != nil {
		return
	}
	return
}

func Convert(fpath string, cs *ConvStat) (err error) {
	var file File
	if file, err = os.Open(fpath); err != nil {
		return // can not open file
	}
	defer file.Close()

	// lazy decode
	var orientation = OrientNormal
	var src image.Image
	var decode = func() {
		if src != nil {
			return
		}

		var imc image.Config
		if imc, _, err = image.DecodeConfig(file); err != nil {
			return // can not recognize format or decode config
		}
		if float32(imc.Width*imc.Height+5e5)/1e6 > Cfg.ImageMaxMpx {
			err = ErrTooBig
			return // file is too big
		}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return // can not seek to start
		}
		if x, err := exif.Decode(file); err == nil {
			var t *tiff.Tag
			if t, err = x.Get(exif.Orientation); err == nil {
				orientation, _ = t.Int(0)
			}
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			return // can not seek to start
		}
		if src, _, err = image.Decode(file); err != nil {
			if src == nil { // skip "short Huffman data" or others errors with partial results
				return // can not decode file by any codec
			}
		}
	}

	var md MediaData
	var size int64
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
		size = fi.Size()
	}

	var ext = GetFileExt(fpath)
	if IsTypeTileImg(ext) && size > 512*1024 {
		for _, tm := range tilemult {
			var wdh, hgt = tm * 24, tm * 18
			var tilepath = fmt.Sprintf("%s?%dx%d", fpath, wdh, hgt)
			if !TilesPkg.HasTagset(tilepath) {
				if decode(); err != nil {
					return
				}
				if md.Data, err = DrawTile(src, wdh, hgt, orientation); err != nil {
					return
				}
				// push tile to package
				if err = TilesPkg.PutFile(tilepath, md); err != nil {
					return
				}
				atomic.AddUint64(&cs.tilecount, 1)
				atomic.AddUint64(&cs.tilesize, uint64(len(md.Data)))
			}
		}
	}

	if !ThumbPkg.HasTagset(fpath) {
		if decode(); err != nil {
			return
		}
		if md.Data, err = DrawThumb(src, orientation); err != nil {
			return
		}
		// push thumbnail to package
		if err = ThumbPkg.PutFile(fpath, md); err != nil {
			return
		}
		atomic.AddUint64(&cs.tmbcount, 1)
		atomic.AddUint64(&cs.tmbsize, uint64(len(md.Data)))
	}

	atomic.AddUint64(&cs.filecount, 1)
	atomic.AddUint64(&cs.filesize, uint64(size))

	return
}

func BatchCacher(list FileMap) {
	var errcount int64
	var cs ConvStat
	var thrnum = GetScanThreadsNum()

	fmt.Fprintf(os.Stdout, "start processing %d files with %d threads...\n", len(list), thrnum)
	var t0 = time.Now()

	// manager thread that distributes task portions
	var cpath = make(chan string, thrnum)
	go func() {
		defer close(cpath)

		var total = float64(len(list))
		var tsync = time.NewTicker(4 * time.Minute)
		defer tsync.Stop()
		var tinfo = time.NewTicker(time.Second)
		defer tinfo.Stop()
		for s := range list {
			select {
			case cpath <- s:
				continue
			case <-tsync.C:
				// sync file tags tables of caches
				if err := ThumbPkg.Sync(); err != nil {
					Log.Error(err)
					return
				}
				if err := TilesPkg.Sync(); err != nil {
					Log.Error(err)
					return
				}
			case <-tinfo.C:
				// information thread
				if ready := atomic.LoadUint64(&cs.filecount); ready > 0 {
					var remain = time.Duration(float64(time.Now().Sub(t0)) / float64(ready) * (total - float64(ready)))
					if remain < time.Hour {
						remain = remain / time.Second * time.Second
					} else {
						remain = remain / time.Minute * time.Minute
					}
					fmt.Fprintf(os.Stdout, "processed %d files, remains about %v            \r",
						ready, remain)
				}
			case <-exitctx.Done():
				return
			}
		}
	}()

	// working threads, runs until 'cpath' not closed
	var workwg sync.WaitGroup
	workwg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		go func() {
			defer workwg.Done()
			for fpath := range cpath {
				var err error
				if err = Convert(fpath, &cs); err != nil {
					atomic.AddInt64(&errcount, 1)
				}
			}
		}()
	}
	workwg.Wait()

	var d = time.Now().Sub(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "processed %d files, spent %v, processing complete\n", cs.filecount, d)
	fmt.Fprintf(os.Stdout, "produced %d tiles and %d thumbnails\n", cs.tilecount, cs.tmbcount)
	if cs.tilesize > 0 {
		fmt.Fprintf(os.Stdout, "tiles size: %d, ratio: %.4f\n", cs.tilesize, float64(cs.filesize)/float64(cs.tilesize))
	}
	if cs.tmbsize > 0 {
		fmt.Fprintf(os.Stdout, "thumbnails size: %d, ratio: %.4f\n", cs.tmbsize, float64(cs.filesize)/float64(cs.tmbsize))
	}
	if errcount > 0 {
		fmt.Fprintf(os.Stdout, "gets %d failures on files\n", errcount)
	}
}

func RunCacher() {
	var shares []string
	for _, acc := range PrfList {
		for _, shr := range acc.Shares {
			var add = true
			for i, p := range shares {
				if strings.HasPrefix(shr.Path, p) {
					add = false
					break
				}
				if strings.HasPrefix(p, shr.Path) {
					shares[i] = shr.Path
					add = false
					break
				}
			}
			if add {
				if _, ok := CatPathKey[shr.Path]; !ok {
					shares = append(shares, shr.Path)
				}
			}
		}
	}
	for _, fpath := range Cfg.ExceptPath {
		for i, p := range shares {
			if strings.HasPrefix(p, fpath) {
				shares = append(shares[:i], shares[i+1:]...)
				break
			}
		}
	}
	for _, fpath := range Cfg.CacherPath {
		var add = true
		for i, p := range shares {
			if strings.HasPrefix(fpath, p) {
				add = false
				break
			}
			if strings.HasPrefix(p, fpath) {
				shares[i] = fpath
				add = false
				break
			}
		}
		if add {
			if _, ok := CatPathKey[fpath]; !ok {
				shares = append(shares, fpath)
			}
		}
	}

	fmt.Fprintf(os.Stdout, "found %d unical shares\n", len(shares))
	var list = FileMap{}
	var sum int
	for i, p := range shares {
		fmt.Fprintf(os.Stdout, "scan %d share with path %s\n", i+1, p)
		var t0 = time.Now()
		if err := FileList(FS(p), list); err != nil {
			log.Fatal(err)
		}
		var d = time.Now().Sub(t0)
		fmt.Fprintf(os.Stdout, "found %d files to process, spent %v\n", len(list)-sum, d)
		sum = len(list)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	BatchCacher(list)
}

// The End.
