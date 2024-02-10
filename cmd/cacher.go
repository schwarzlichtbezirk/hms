package cmd

import (
	"context"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	srv "github.com/schwarzlichtbezirk/hms/server"
	jnt "github.com/schwarzlichtbezirk/joint"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

type FileMap = map[string]fs.FileInfo

type CnvStat struct {
	ErrCount  uint64
	FileCount uint64
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

type pathpair struct {
	fpath string
	fi    fs.FileInfo
}

// IsCached returns "true" if thumbnail and all tiles are cached for given filename.
func IsCached(fpath string) bool {
	if !srv.ThumbPkg.HasTagset(fpath) {
		return false
	}

	for _, tm := range tilemult {
		var wdh, hgt = tm * 24, tm * 18
		var tilepath = fmt.Sprintf("%s?%dx%d", fpath, wdh, hgt)
		if !srv.TilesPkg.HasTagset(tilepath) {
			return false
		}
	}

	return true
}

// FileList forms list of files to process by caching algorithm.
func FileList(fsys *jnt.SubPool, pathlist *[]string, extlist, cnvlist FileMap) (err error) {
	var session = srv.XormStorage.NewSession()
	defer session.Close()

	fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var fullpath = JoinPath(fsys.Dir(), fpath)
		if d.Name() != "." && d.Name() != ".." {
			if _, ok := srv.PathStorePUID(session, fullpath); !ok {
				*pathlist = append(*pathlist, fullpath)
			}
		}
		if d.IsDir() {
			return nil // file is directory
		}
		var ext = srv.GetFileExt(fpath)
		if srv.IsTypeDecoded(ext) {
			if !IsCached(fullpath) {
				cnvlist[fullpath], _ = d.Info()
			}
		}
		if srv.IsTypeEXIF(ext) || srv.IsTypeDecoded(ext) || srv.IsTypeID3(ext) {
			extlist[fullpath], _ = d.Info()
		}
		return nil
	})
	return
}

func Convert(fpath string, fi fs.FileInfo, cs *CnvStat) (err error) {
	defer func() {
		if err != nil {
			atomic.AddUint64(&cs.ErrCount, 1)
		}
	}()

	// lazy decode
	var orientation = srv.OrientNormal
	var src image.Image
	var imc image.Config
	var decode = func() {
		if src != nil {
			return
		}

		var file io.ReadSeekCloser
		if file, err = os.Open(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		if imc, _, err = image.DecodeConfig(file); err != nil {
			return // can not recognize format or decode config
		}
		if float32(imc.Width*imc.Height+5e5)/1e6 > Cfg.ImageMaxMpx {
			err = srv.ErrTooBig
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

	var md srv.MediaData
	md.Mime = srv.MimeWebp
	md.Time = fi.ModTime()
	var size = fi.Size()

	atomic.AddUint64(&cs.FileCount, 1)
	atomic.AddUint64(&cs.filesize, uint64(size))

	var ext = srv.GetFileExt(fpath)
	if srv.IsTypeTileImg(ext) && size > 512*1024 {
		for _, tm := range tilemult {
			var wdh, hgt = tm * 24, tm * 18
			var tilepath = fmt.Sprintf("%s?%dx%d", fpath, wdh, hgt)
			if !srv.TilesPkg.HasTagset(tilepath) {
				if decode(); err != nil {
					return
				}
				if md.Data, err = srv.DrawTile(src, wdh, hgt, orientation); err != nil {
					return
				}
				// push tile to package
				if err = srv.TilesPkg.PutFile(tilepath, md); err != nil {
					return
				}
				atomic.AddUint64(&cs.tilecount, 1)
				atomic.AddUint64(&cs.tilesize, uint64(len(md.Data)))
			}
		}
	}

	if !srv.ThumbPkg.HasTagset(fpath) {
		if decode(); err != nil {
			return
		}
		if md.Data, err = srv.DrawThumb(src, imc.Width, imc.Height, orientation); err != nil {
			return
		}
		// push thumbnail to package
		if err = srv.ThumbPkg.PutFile(fpath, md); err != nil {
			return
		}
		atomic.AddUint64(&cs.tmbcount, 1)
		atomic.AddUint64(&cs.tmbsize, uint64(len(md.Data)))
	}

	return
}

func BatchPathList(pathlist []string) {
	var err error
	fmt.Fprintf(os.Stdout, "start caching %d file paths\n", len(pathlist))
	var t0 = time.Now()

	var session = srv.XormStorage.NewSession()
	defer session.Close()

	const limit = 256
	for i := 0; i < len(pathlist); i += limit {
		var pc []string
		if len(pathlist)-i >= limit {
			pc = pathlist[i : i+limit]
		} else {
			pc = pathlist[i:]
		}
		var nps = make([]srv.PathStore, len(pc))
		for i, fpath := range pc {
			nps[i].Path = fpath
			nps[i].Puid = 0
		}
		if _, err = session.Insert(&nps); err != nil {
			fmt.Fprintf(os.Stdout, "error received: %s\n", err.Error())
		}
		nps = nil
		if err = session.In("path", pc).Find(&nps); err != nil {
			return
		}
		for _, ps := range nps {
			if ps.Puid != 0 {
				srv.PathCache.Set(ps.Puid, ps.Path)
			}
		}
	}

	var d = time.Since(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "file paths caching complete, spent %v\n", d)
}

func UpdateExtList(extlist FileMap) {
	var session = srv.XormStorage.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var puids []uint64
		if err := session.Table("ext_store").Cols("puid").Limit(limit, offset).Find(&puids); err != nil {
			return
		}
		offset += limit
		for _, puid := range puids {
			if fpath, ok := srv.PathStorePath(session, srv.Puid_t(puid)); ok {
				delete(extlist, fpath)
			} else {
				fmt.Fprintf(os.Stdout, "found unlinked PUID in ext_store: %d\n", puid)
			}
		}
		if limit > len(puids) {
			break
		}
	}
}

func BatchExtractor(exitctx context.Context, extlist FileMap) {
	var es srv.ExtStat
	var thrnum = srv.GetScanThreadsNum()

	fmt.Fprintf(os.Stdout, "start processing %d files with %d threads...\n", len(extlist), thrnum)
	var t0 = time.Now()

	// manager thread that distributes task to extract emedded information
	var pathchan = make(chan pathpair, thrnum)
	go func() {
		defer close(pathchan)

		var total = float64(len(extlist))
		var tinfo = time.NewTicker(time.Second)
		defer tinfo.Stop()
		for fpath, fi := range extlist {
			select {
			case pathchan <- pathpair{fpath, fi}:
				continue
			case <-tinfo.C:
				// information thread
				if ready := atomic.LoadUint64(&es.FileCount); ready > 0 {
					var remain = time.Duration(float64(time.Since(t0)) / float64(ready) * (total - float64(ready)))
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

	// working threads, runs until 'pathchan' not closed
	var workwg sync.WaitGroup
	workwg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		go func() {
			defer workwg.Done()

			var session = srv.XormStorage.NewSession()
			defer session.Close()

			var buf srv.StoreBuf
			buf.Init(256)
			defer buf.Flush(session)

			for c := range pathchan {
				srv.TagsExtract(c.fpath, session, &buf, &es, false)
			}
		}()
	}
	workwg.Wait()

	var d = time.Since(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "processed %d files, spent %v, processing complete\n", es.FileCount, d)
	fmt.Fprintf(os.Stdout, "total %d files with embedded info processed, %d of them with EXIF, %d of them with ID3 tags, %d embedded thumbnails, %d mp3-files\n",
		es.ExtCount, es.ExifCount, es.Id3Count, es.TmbCount, es.Mp3Count)
	if es.ErrCount > 0 {
		fmt.Fprintf(os.Stdout, "gets %d failures on embedded info extract\n", es.ErrCount)
	}
}

func BatchCacher(exitctx context.Context, cnvlist FileMap) {
	var cs CnvStat
	var thrnum = srv.GetScanThreadsNum()

	fmt.Fprintf(os.Stdout, "start processing %d files with %d threads to prepare tiles and thumbnails...\n", len(cnvlist), thrnum)
	var t0 = time.Now()

	// manager thread that distributes task to convert images
	var pathchan = make(chan pathpair, thrnum)
	go func() {
		defer close(pathchan)

		var total = float64(len(cnvlist))
		var tsync = time.NewTicker(4 * time.Minute)
		defer tsync.Stop()
		var tinfo = time.NewTicker(time.Second)
		defer tinfo.Stop()
		for fpath, fi := range cnvlist {
			select {
			case pathchan <- pathpair{fpath, fi}:
				continue
			case <-tsync.C:
				// sync file tags tables of caches
				if err := srv.ThumbPkg.Sync(); err != nil {
					Log.Error(err)
					return
				}
				if err := srv.TilesPkg.Sync(); err != nil {
					Log.Error(err)
					return
				}
			case <-tinfo.C:
				// information thread
				if ready := atomic.LoadUint64(&cs.FileCount); ready > 0 {
					var remain = time.Duration(float64(time.Since(t0)) / float64(ready) * (total - float64(ready)))
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

	// working threads, runs until 'pathchan' not closed
	var workwg sync.WaitGroup
	workwg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		go func() {
			defer workwg.Done()

			for c := range pathchan {
				Convert(c.fpath, c.fi, &cs)
			}
		}()
	}
	workwg.Wait()

	var d = time.Since(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "processed %d files, spent %v, processing complete\n", cs.FileCount, d)
	fmt.Fprintf(os.Stdout, "produced %d tiles and %d thumbnails\n", cs.tilecount, cs.tmbcount)
	if cs.tilesize > 0 {
		fmt.Fprintf(os.Stdout, "tiles size: %d, ratio: %.4f\n", cs.tilesize, float64(cs.filesize)/float64(cs.tilesize))
	}
	if cs.tmbsize > 0 {
		fmt.Fprintf(os.Stdout, "thumbnails size: %d, ratio: %.4f\n", cs.tmbsize, float64(cs.filesize)/float64(cs.tmbsize))
	}
	if cs.ErrCount > 0 {
		fmt.Fprintf(os.Stdout, "gets %d failures on image conversions\n", cs.ErrCount)
	}
}

func RunCacher(exitctx context.Context) {
	fmt.Fprintf(os.Stdout, "starts caching processing\n")

	var shares []string
	srv.Profiles.Range(func(id uint64, prf *srv.Profile) bool {
		for _, shr := range prf.Shares {
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
				if _, ok := srv.CatPathKey[shr.Path]; !ok {
					shares = append(shares, shr.Path)
				}
			}
		}
		return true
	})
	for _, fpath := range ExcludePath {
		for i, p := range shares {
			if strings.HasPrefix(p, fpath) {
				shares = append(shares[:i], shares[i+1:]...)
				break
			}
		}
	}
	for _, fpath := range IncludePath {
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
			if _, ok := srv.CatPathKey[fpath]; !ok {
				shares = append(shares, fpath)
			}
		}
	}

	fmt.Fprintf(os.Stdout, "found %d unical shares\n", len(shares))
	var pathlist []string
	var extlist, cnvlist = FileMap{}, FileMap{}
	var extsum, cnvsum int
	for i, p := range shares {
		fmt.Fprintf(os.Stdout, "starts scan %d share with path %s\n", i+1, p)
		var t0 = time.Now()
		var err error
		var sub fs.FS
		if sub, err = srv.JP.Sub(p); err != nil {
			Log.Fatal(err)
		}
		if err = FileList(sub.(*jnt.SubPool), &pathlist, extlist, cnvlist); err != nil {
			Log.Fatal(err)
		}
		var d = time.Since(t0)
		fmt.Fprintf(os.Stdout, "scan %d share complete, spent %v\n", i+1, d)
		fmt.Fprintf(os.Stdout, "found %d files to extract embedded info\n", len(extlist)-extsum)
		extsum = len(extlist)
		fmt.Fprintf(os.Stdout, "found %d files to prepare tiles and thumbnails\n", len(cnvlist)-cnvsum)
		cnvsum = len(cnvlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	if len(pathlist) > 0 {
		BatchPathList(pathlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	UpdateExtList(extlist)

	if len(extlist) > 0 {
		BatchExtractor(exitctx, extlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	if len(cnvlist) > 0 {
		BatchCacher(exitctx, cnvlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	fmt.Fprintf(os.Stdout, "all processing complete\n")
}

// The End.
