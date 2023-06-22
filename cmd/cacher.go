package main

import (
	"context"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/schwarzlichtbezirk/hms"
	. "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

type FileMap = map[string]struct{}

func FileList(fsys FS, list FileMap) (err error) {
	fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var fi fs.FileInfo
		if fi, err = d.Info(); err != nil {
			return nil
		}
		if fi.IsDir() {
			return nil // file is directory
		}
		var ext = GetFileExt(fpath)
		if !IsTypeImage(ext) {
			return nil // file is not image
		}
		if !CheckImageSize(ext, fi.Size()) {
			return nil // file is too big
		}
		list[path.Join(string(fsys), fpath)] = struct{}{}
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

func Convert(fpath string, cs *ConvStat) (err error) {
	var file *os.File
	if file, err = os.Open(fpath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var fi fs.FileInfo
	if fi, err = file.Stat(); err != nil {
		return
	}
	if fi.IsDir() {
		err = ErrNotFile // file is directory
		return
	}

	var ext = GetFileExt(fpath)

	// check that file is image
	if !IsTypeImage(ext) {
		err = ErrNotImg
		return // file is not image
	}

	if !CheckImageSize(ext, fi.Size()) {
		err = ErrTooBig
		return // file is too big
	}

	// lazy decode
	var orientation = OrientNormal
	var src image.Image
	var decode = func() {
		if src != nil {
			return
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
	md.Mime = MimeWebp
	if fi, _ := file.Stat(); fi != nil {
		md.Time = fi.ModTime()
	}

	if (ext == ".jpg" || ext == ".webp") && fi.Size() > 512*1024 {
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
	atomic.AddUint64(&cs.filesize, uint64(fi.Size()))

	return
}

func BatchCacher(list FileMap) {
	var errcount int64
	var cs ConvStat
	var thrnum = GetScanThreadsNum()

	// manager thread that distributes task portions
	var cpath = make(chan string, thrnum)
	go func() {
		defer close(cpath)
		for s := range list {
			select {
			case cpath <- s:
				continue
			case <-exitctx.Done():
				return
			}
		}
	}()

	// working threads, runs until 'cpath' not closed
	fmt.Fprintf(os.Stdout, "start processing %d files with %d threads...\n", len(list), thrnum)
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

	// information thread
	var ctx, cancel = context.WithCancel(context.Background())
	var infowg sync.WaitGroup
	infowg.Add(1)
	go func() {
		defer infowg.Done()

		var t0 = time.Now()
		for {
			select {
			case <-ctx.Done():
				var d = time.Now().Sub(t0) / time.Second * time.Second
				fmt.Fprintf(os.Stdout, "processed %d files, spent %v, processing complete\n", cs.filecount, d)
				fmt.Fprintf(os.Stdout, "produced %d tiles and %d thumbnails\n", cs.tilecount, cs.tmbcount)
				if cs.tilesize > 0 {
					fmt.Fprintf(os.Stdout, "tiles size: %d, ratio: %.4g\n", cs.tilesize, float64(cs.filesize)/float64(cs.tilesize))
				}
				if cs.tmbsize > 0 {
					fmt.Fprintf(os.Stdout, "thumbnails size: %d, ratio: %.4g\n", cs.tmbsize, float64(cs.filesize)/float64(cs.tmbsize))
				}
				if errcount > 0 {
					fmt.Fprintf(os.Stdout, "gets %d failures on files\n", errcount)
				}
				return
			case <-time.After(time.Second):
				var ready = atomic.LoadUint64(&cs.filecount)
				var d time.Duration
				if ready > 0 {
					d = time.Duration(float64(time.Now().Sub(t0))/float64(ready)*float64(uint64(len(list))-ready)) / time.Second * time.Second
				}
				fmt.Fprintf(os.Stdout, "processed %d files, remains about %v            \r", ready, d)
			}
		}
	}()

	workwg.Wait()
	cancel()
	infowg.Wait()
}

func RunCacher() {
	var shares []string
	for _, acc := range PrfList {
	prfshr:
		for _, shr := range acc.Shares {
			for i, p := range shares {
				if strings.HasPrefix(shr.Path, p) {
					continue prfshr
				}
				if strings.HasPrefix(p, shr.Path) {
					shares[i] = shr.Path
					continue prfshr
				}
			}
			if _, ok := CatPathKey[shr.Path]; !ok {
				shares = append(shares, shr.Path)
			}
		}
	}

	fmt.Fprintf(os.Stdout, "found %d unical shares\n", len(shares))
	var list = FileMap{}
	var sum int
	for i, p := range shares {
		fmt.Fprintf(os.Stdout, "scan %d share with path %s\n", i+1, p)
		if err := FileList(FS(p), list); err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stdout, "found %d files to process\n", len(list)-sum)
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
