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
	"sync"
	"sync/atomic"
	"time"

	. "github.com/schwarzlichtbezirk/hms"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

func FileList(fsys fs.FS) (list []string, err error) {
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
		if !IsTypeImage(GetFileExt(fpath)) {
			return nil // file is not image
		}
		if fi.Size() > 50*1024*1024 /*Cfg.ThumbFileMaxSize*/ {
			return nil // file is too big
		}
		list = append(list, fpath)
		return nil
	})
	return
}

// Tiles multipliers:
var tilemult = [...]int{
	2, 3, 4, 6, 8, 9, 10, 12, 15, 16, 18, 20, 24, 30, 36,
}

func Convert(fpath string) (count int, err error) {
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

	if fi.Size() > 50*1024*1024 /*Cfg.ThumbFileMaxSize*/ {
		err = ErrTooBig
		return // file is too big
	}

	var orientation = OrientNormal
	if x, err := exif.Decode(file); err == nil {
		var t *tiff.Tag
		if t, err = x.Get(exif.Orientation); err == nil {
			orientation, _ = t.Int(0)
		}
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return // can not seek to start
	}

	var src image.Image
	if src, _, err = image.Decode(file); err != nil {
		if src == nil { // skip "short Huffman data" or others errors with partial results
			return // can not decode file by any codec
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
				if md.Data, err = DrawTile(src, wdh, hgt, orientation); err != nil {
					return
				}
				// push tile to package
				if err = TilesPkg.PutFile(tilepath, md); err != nil {
					return
				}
				count++
			}
		}
	}

	if !ThumbPkg.HasTagset(fpath) {
		if md.Data, err = DrawThumb(src, orientation); err != nil {
			return
		}
		// push thumbnail to package
		if err = ThumbPkg.PutFile(fpath, md); err != nil {
			return
		}
		count++
	}

	return
}

var root = "D:/test/imggps"

func RunCacher() {
	var err error
	var list []string
	var total, ready, drawcount, errcount int64

	var wg sync.WaitGroup
	defer wg.Wait()

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				fmt.Fprintf(os.Stdout, "processing complete            \n")
				fmt.Fprintf(os.Stdout, "produced %d tiles and thumbnails\n", drawcount)
				if errcount > 0 {
					fmt.Fprintf(os.Stdout, "gets %d failures on files\n", errcount)
				}
				return
			case <-time.After(time.Second):
				fmt.Fprintf(os.Stdout, "processed %d files\r", atomic.LoadInt64(&ready))
			}
		}
	}()

	var cpath = make(chan string)
	go func() {
		for len(list) > 0 {
			var s = list[len(list)-1]
			list = list[:len(list)-1]
			select {
			case cpath <- s:
			}
		}
		close(cpath)
	}()

	if list, err = FileList(os.DirFS(root)); err != nil {
		log.Fatal(err)
	}
	total = int64(len(list))
	fmt.Fprintf(os.Stdout, "total files to process: %d\n", total)
	for fpath := range cpath {
		var count int
		if count, err = Convert(path.Join(root, fpath)); err != nil {
			atomic.AddInt64(&errcount, 1)
		}
		atomic.AddInt64(&drawcount, int64(count))
		atomic.AddInt64(&ready, 1)
	}
}

// The End.
