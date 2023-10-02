package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhowden/tag"
	. "github.com/schwarzlichtbezirk/hms"
	. "github.com/schwarzlichtbezirk/hms/config"
	. "github.com/schwarzlichtbezirk/hms/joint"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"xorm.io/xorm"
)

type FileMap = map[string]fs.FileInfo

type ExtStat struct {
	errcount  uint64
	filecount uint64
	extcount  uint64
	exifcount uint64
	id3count  uint64
}

type CnvStat struct {
	errcount  uint64
	filecount uint64
	tilecount uint64
	tmbcount  uint64
	filesize  uint64
	tilesize  uint64
	tmbsize   uint64
}

type ExtBuf struct {
	extbuf  []ExtStore
	exifbuf []ExifStore
	id3buf  []TagStore
}

var (
	ErrBadType = errors.New("type does not supported to insert into database")
)

func (buf *ExtBuf) Init() {
	const limit = 256
	buf.extbuf = make([]ExtStore, 0, limit)
	buf.exifbuf = make([]ExifStore, 0, limit)
	buf.id3buf = make([]TagStore, 0, limit)
}

func (buf *ExtBuf) Push(val any) {
	switch st := val.(type) {
	case ExtStore:
		buf.extbuf = append(buf.extbuf, st)
	case ExifStore:
		buf.exifbuf = append(buf.exifbuf, st)
	case TagStore:
		buf.id3buf = append(buf.id3buf, st)
	default:
		panic(ErrBadType)
	}
}

func (buf *ExtBuf) Overflow(session *xorm.Session) (err error) {
	if len(buf.extbuf) == cap(buf.extbuf) {
		if _, err = session.Insert(&buf.extbuf); err != nil {
			return
		}
		buf.extbuf = buf.extbuf[:0]
	}
	if len(buf.exifbuf) == cap(buf.exifbuf) {
		if _, err = session.Insert(&buf.exifbuf); err != nil {
			return
		}
		buf.exifbuf = buf.exifbuf[:0]
	}
	if len(buf.id3buf) == cap(buf.id3buf) {
		if _, err = session.Insert(&buf.id3buf); err != nil {
			return
		}
		buf.id3buf = buf.id3buf[:0]
	}
	return
}

func (buf *ExtBuf) Flush(session *xorm.Session) (err error) {
	if len(buf.extbuf) > 0 {
		if _, err = session.Insert(&buf.extbuf); err != nil {
			return
		}
		buf.extbuf = buf.extbuf[:0]
	}
	if len(buf.exifbuf) > 0 {
		if _, err = session.Insert(&buf.exifbuf); err != nil {
			return
		}
		buf.exifbuf = buf.exifbuf[:0]
	}
	if len(buf.id3buf) > 0 {
		if _, err = session.Insert(&buf.id3buf); err != nil {
			return
		}
		buf.id3buf = buf.id3buf[:0]
	}
	return
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

// FileList forms list of files to process by caching algorithm.
func FileList(fsys FS, pathlist *[]string, extlist, cnvlist FileMap) (err error) {
	fs.WalkDir(fsys, ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var fullpath = JoinFast(string(fsys), fpath)
		if d.Name() != "." && d.Name() != ".." {
			if _, ok := PathCache.GetRev(fullpath); !ok {
				*pathlist = append(*pathlist, fullpath)
			}
		}
		if d.IsDir() {
			return nil // file is directory
		}
		var ext = GetFileExt(fpath)
		if IsTypeDecoded(ext) {
			if !IsCached(fullpath) {
				cnvlist[fullpath], _ = d.Info()
			}
		}
		if IsTypeEXIF(ext) || IsTypeDecoded(ext) || IsTypeID3(ext) {
			extlist[fullpath], _ = d.Info()
		}
		return nil
	})
	return
}

func Extract(fpath string, buf *ExtBuf, es *ExtStat) (err error) {
	atomic.AddUint64(&es.filecount, 1)

	var puid, _ = PathCache.GetRev(fpath)
	var ext = GetFileExt(fpath)
	if IsTypeEXIF(ext) {
		var file File
		if file, err = os.Open(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var xp ExtProp
		var ep ExifProp
		var imc image.Config

		if x, err := exif.Decode(file); err == nil {
			ep.Setup(x)
			if !ep.IsZero() {
				GpsCachePut(puid, ep)
				buf.Push(ExifStore{
					Puid: puid,
					Prop: ep,
				})
				xp.Content = CntExif // EXIF is exist
				atomic.AddUint64(&es.exifcount, 1)
			}
		}

		if _, err = file.Seek(io.SeekStart, 0); err != nil {
			return
		}
		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		xp.Width, xp.Height = imc.Width, imc.Height
		if ep.ThumbJpegLen > 0 {
			xp.Content |= CntThumb
		}
		buf.Push(ExtStore{
			Puid: puid,
			Prop: xp,
		})
		atomic.AddUint64(&es.extcount, 1)
	} else if IsTypeDecoded(ext) {
		var file File
		if file, err = os.Open(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var xp ExtProp
		var imc image.Config

		if imc, _, err = image.DecodeConfig(file); err != nil {
			return
		}
		xp.Content = 0
		xp.Width, xp.Height = imc.Width, imc.Height
		buf.Push(ExtStore{
			Puid: puid,
			Prop: xp,
		})
		atomic.AddUint64(&es.extcount, 1)
	} else if IsTypeID3(ext) {
		var file File
		if file, err = os.Open(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

		var xp ExtProp
		var tp TagProp

		if m, err := tag.ReadFrom(file); err == nil {
			tp.Setup(m)
			if !tp.IsZero() {
				buf.Push(TagStore{
					Puid: puid,
					Prop: tp,
				})
				xp.Content = CntID3
				atomic.AddUint64(&es.id3count, 1)
			}
		}

		if tp.ThumbLen > 0 {
			xp.Content |= CntThumb
		}
		buf.Push(ExtStore{
			Puid: puid,
			Prop: xp,
		})
		atomic.AddUint64(&es.extcount, 1)
	}

	return
}

func Convert(fpath string, fi fs.FileInfo, cs *CnvStat) (err error) {
	// lazy decode
	var orientation = OrientNormal
	var src image.Image
	var decode = func() {
		if src != nil {
			return
		}

		var file File
		if file, err = os.Open(fpath); err != nil {
			return // can not open file
		}
		defer file.Close()

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
	md.Mime = MimeWebp
	md.Time = fi.ModTime()
	var size = fi.Size()

	atomic.AddUint64(&cs.filecount, 1)
	atomic.AddUint64(&cs.filesize, uint64(size))

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

	return
}

func BatchPathList(pathlist []string) {
	var err error
	fmt.Fprintf(os.Stdout, "start caching %d file paths\n", len(pathlist))
	var t0 = time.Now()

	var session = XormStorage.NewSession()
	defer session.Close()

	const limit = 256
	for i := 0; i < len(pathlist); i += limit {
		var pc []string
		if len(pathlist)-i >= limit {
			pc = pathlist[i : i+limit]
		} else {
			pc = pathlist[i:]
		}
		var nps = make([]PathStore, len(pc))
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
				PathCache.Set(ps.Puid, ps.Path)
			}
		}
	}

	var d = time.Now().Sub(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "file paths caching complete, spent %v\n", d)
}

func UpdateExtList(extlist FileMap) {
	var session = XormStorage.NewSession()
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
			if fpath, ok := PathCache.GetDir(Puid_t(puid)); ok {
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

func BatchExtractor(extlist FileMap) {
	var es ExtStat
	var thrnum = GetScanThreadsNum()

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
				if ready := atomic.LoadUint64(&es.filecount); ready > 0 {
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

	// working threads, runs until 'pathchan' not closed
	var workwg sync.WaitGroup
	workwg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		go func() {
			defer workwg.Done()

			var session = XormStorage.NewSession()
			defer session.Close()

			var err error
			var buf ExtBuf
			buf.Init()

			for c := range pathchan {
				if err = Extract(c.fpath, &buf, &es); err != nil {
					atomic.AddUint64(&es.errcount, 1)
					fmt.Fprintf(os.Stdout, "error on file %s: %s\n", c.fpath, err.Error())
				}
				buf.Overflow(session)
			}
			buf.Flush(session)
		}()
	}
	workwg.Wait()

	var d = time.Now().Sub(t0) / time.Second * time.Second
	fmt.Fprintf(os.Stdout, "processed %d files, spent %v, processing complete\n", es.filecount, d)
	fmt.Fprintf(os.Stdout, "total %d files with embedded info processed, %d of them with EXIF, %d of them with ID3 tags\n", es.extcount, es.exifcount, es.id3count)
	if es.errcount > 0 {
		fmt.Fprintf(os.Stdout, "gets %d failures on embedded info extract\n", es.errcount)
	}
}

func BatchCacher(cnvlist FileMap) {
	var cs CnvStat
	var thrnum = GetScanThreadsNum()

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

	// working threads, runs until 'pathchan' not closed
	var workwg sync.WaitGroup
	workwg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		go func() {
			defer workwg.Done()

			var err error
			for c := range pathchan {
				if err = Convert(c.fpath, c.fi, &cs); err != nil {
					atomic.AddUint64(&cs.errcount, 1)
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
	if cs.errcount > 0 {
		fmt.Fprintf(os.Stdout, "gets %d failures on image conversions\n", cs.errcount)
	}
}

func RunCacher() {
	fmt.Fprintf(os.Stdout, "starts caching processing\n")

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
	var pathlist []string
	var extlist, cnvlist = FileMap{}, FileMap{}
	var extsum, cnvsum int
	for i, p := range shares {
		fmt.Fprintf(os.Stdout, "scan %d share with path %s\n", i+1, p)
		var t0 = time.Now()
		if err := FileList(FS(p), &pathlist, extlist, cnvlist); err != nil {
			Log.Fatal(err)
		}
		var d = time.Now().Sub(t0)
		fmt.Fprintf(os.Stdout, "found %d files to extract embedded info, spent %v\n", len(extlist)-extsum, d)
		extsum = len(extlist)
		fmt.Fprintf(os.Stdout, "found %d files to prepare tiles and thumbnails, spent %v\n", len(cnvlist)-cnvsum, d)
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
		BatchExtractor(extlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	if len(cnvlist) > 0 {
		BatchCacher(cnvlist)

		select {
		case <-exitctx.Done():
			return
		default:
		}
	}

	fmt.Fprintf(os.Stdout, "all processing complete\n")
}

// The End.
