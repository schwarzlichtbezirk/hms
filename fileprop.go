package hms

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// File types
const (
	FTctgr = -3
	FTdrv  = -2
	FTdir  = -1
	FTfile = 0
	FTmp4  = 1
	FTwebm = 2
	FTmov  = 3
	FTwave = 4
	FTflac = 5
	FTmus  = 6
	FTtga  = 7
	FTbmp  = 8
	FTdds  = 9
	FTtiff = 10
	FTjpeg = 11
	FTgif  = 12
	FTpng  = 13
	FTwebp = 14
	FTpsd  = 15
	FTimg  = 16
	FTpdf  = 17
	FThtml = 18
	FTtext = 19
	FTscr  = 20
	FTcfg  = 21
	FTlog  = 22
	FTarch = 23
	FTdisk = 24
	FTpack = 25
)

// File groups
const (
	FGother = 0
	FGvideo = 1
	FGaudio = 2
	FGimage = 3
	FGbooks = 4
	FGtexts = 5
	FGdisks = 6
	FGdir   = 7
)

// FGnum is count of file groups
const FGnum = 8

var typetogroup = map[int]int{
	FTctgr: FGdir,
	FTdrv:  FGdir,
	FTdir:  FGdir,
	FTfile: FGother,
	FTmp4:  FGvideo,
	FTwebm: FGvideo,
	FTmov:  FGvideo,
	FTwave: FGaudio,
	FTflac: FGaudio,
	FTmus:  FGaudio,
	FTtga:  FGimage,
	FTbmp:  FGimage,
	FTdds:  FGimage,
	FTtiff: FGimage,
	FTjpeg: FGimage,
	FTgif:  FGimage,
	FTpng:  FGimage,
	FTwebp: FGimage,
	FTpsd:  FGimage,
	FTimg:  FGimage,
	FTpdf:  FGbooks,
	FThtml: FGbooks,
	FTtext: FGtexts,
	FTscr:  FGtexts,
	FTcfg:  FGtexts,
	FTlog:  FGtexts,
	FTarch: FGdisks,
	FTdisk: FGdisks,
	FTpack: FGdisks,
}

var extset = map[string]int{
	// Video
	".mp4":  FTmp4,
	".webm": FTwebm,
	".mov":  FTmov,
	".avi":  FTmov,
	".mkv":  FTmov,

	// Audio
	".wav":  FTwave,
	".flac": FTflac,
	".mp3":  FTmus,
	".ogg":  FTmus,
	".opus": FTmus,
	".acc":  FTmus,
	".m4a":  FTmus,
	".wma":  FTmus,

	// Images
	".tga":  FTtga,
	".bmp":  FTbmp,
	".dib":  FTbmp,
	".rle":  FTbmp,
	".dds":  FTdds,
	".tif":  FTtiff,
	".tiff": FTtiff,
	".jpg":  FTjpeg,
	".jpe":  FTjpeg,
	".jpeg": FTjpeg,
	".jfif": FTjpeg,
	".gif":  FTgif,
	".png":  FTpng,
	".webp": FTwebp,
	".psd":  FTpsd,
	".psb":  FTpsd,
	".jp2":  FTimg,
	".jpg2": FTimg,
	".jpx":  FTimg,
	".jxr":  FTimg,

	// Books
	".pdf":   FTpdf,
	".html":  FThtml,
	".htm":   FThtml,
	".shtml": FThtml,
	".shtm":  FThtml,
	".xhtml": FThtml,
	".phtml": FThtml,
	".hta":   FThtml,
	".mht":   FThtml,

	// Texts
	".txt":   FTtext,
	".md":    FTtext,
	".css":   FTscr,
	".js":    FTscr,
	".jsm":   FTscr,
	".vb":    FTscr,
	".vbs":   FTscr,
	".bat":   FTscr,
	".cmd":   FTscr,
	".sh":    FTscr,
	".mak":   FTscr,
	".iss":   FTscr,
	".nsi":   FTscr,
	".nsh":   FTscr,
	".bsh":   FTscr,
	".sql":   FTscr,
	".as":    FTscr,
	".mx":    FTscr,
	".php":   FTscr,
	".phpt":  FTscr,
	".java":  FTscr,
	".jsp":   FTscr,
	".asp":   FTscr,
	".lua":   FTscr,
	".tcl":   FTscr,
	".asm":   FTscr,
	".c":     FTscr,
	".h":     FTscr,
	".hpp":   FTscr,
	".hxx":   FTscr,
	".cpp":   FTscr,
	".cxx":   FTscr,
	".cc":    FTscr,
	".cs":    FTscr,
	".go":    FTscr,
	".r":     FTscr,
	".d":     FTscr,
	".pas":   FTscr,
	".inc":   FTscr,
	".py":    FTscr,
	".pyw":   FTscr,
	".pl":    FTscr,
	".pm":    FTscr,
	".plx":   FTscr,
	".rb":    FTscr,
	".rbw":   FTscr,
	".rc":    FTscr,
	".ps":    FTscr,
	".cfg":   FTcfg,
	".ini":   FTcfg,
	".inf":   FTcfg,
	".reg":   FTcfg,
	".url":   FTcfg,
	".xml":   FTcfg,
	".xsml":  FTcfg,
	".xsl":   FTcfg,
	".xsd":   FTcfg,
	".kml":   FTcfg,
	".wsdl":  FTcfg,
	".xlf":   FTcfg,
	".xliff": FTcfg,
	".yml":   FTcfg,
	".yaml":  FTcfg,
	".cmake": FTcfg,
	".json":  FTcfg,
	".log":   FTlog,

	// storage
	".cab":  FTarch,
	".zip":  FTarch,
	".7z":   FTarch,
	".rar":  FTarch,
	".rev":  FTarch,
	".tar":  FTarch,
	".tgz":  FTarch,
	".gz":   FTarch,
	".bz2":  FTarch,
	".iso":  FTdisk,
	".isz":  FTdisk,
	".udf":  FTdisk,
	".nrg":  FTdisk,
	".mdf":  FTdisk,
	".mdx":  FTdisk,
	".img":  FTdisk,
	".ima":  FTdisk,
	".imz":  FTdisk,
	".ccd":  FTdisk,
	".vc4":  FTdisk,
	".dmg":  FTdisk,
	".daa":  FTdisk,
	".uif":  FTdisk,
	".vhd":  FTdisk,
	".vhdx": FTdisk,
	".vmdk": FTdisk,
	".wpk":  FTpack,
}

// Categories properties constants.
const (
	CPhome   = "[home/Home]"
	CPdrives = "[drives/Drives list]"
	CPshares = "[shares/Shared resources]"
	CPmedia  = "[media/Multimedia files]"
	CPvideo  = "[video/Movie and video files]"
	CPaudio  = "[audio/Music and audio files]"
	CPimage  = "[image/Photos and images]"
	CPbooks  = "[books/Books]"
	CPtexts  = "[texts/Text files]"
)

// CatPath is paths list of categories properties.
var CatPath = []string{
	CPhome,
	CPdrives,
	CPshares,
	CPmedia,
	CPvideo,
	CPaudio,
	CPimage,
	CPbooks,
	CPtexts,
}

// CidCatPath is map where key is CID, value is categories paths.
var CidCatPath = map[string]string{
	"home":   CPhome,
	"drives": CPdrives,
	"shares": CPshares,
	"media":  CPmedia,
	"video":  CPvideo,
	"audio":  CPaudio,
	"image":  CPimage,
	"books":  CPbooks,
	"texts":  CPtexts,
}

// GetFileType returns file type integer value for given file name by it's extension.
func GetFileType(fpath string) int {
	return extset[strings.ToLower(path.Ext(fpath))]
}

// Pather is path properties interface.
type Pather interface {
	Name() string // string identifier
	Type() int    // type identifier
	Size() int64  // size in bytes
	Time() int64  // UNIX time in milliseconds
	PUID() string // path unique ID encoded to hex-base32
	NTmb() int    // -1 - can not make thumbnail; 0 - not cached; 1 - cached
	SetNTmb(int)
}

// PathProp is any path base properties.
type PathProp struct {
	NameVal string `json:"name,omitempty" yaml:"name,omitempty"`
	TypeVal int    `json:"type,omitempty" yaml:"type,omitempty"`
}

// Name is file name with extension without path.
func (pp *PathProp) Name() string {
	return pp.NameVal
}

// Type is enumerated file type.
func (pp *PathProp) Type() int {
	return pp.TypeVal
}

// Size is file size in bytes.
func (pp *PathProp) Size() int64 {
	return 0
}

// Time is file creation time in UNIX format, milliseconds.
func (pp *PathProp) Time() int64 {
	return 0
}

// FileProp is common file properties chunk.
type FileProp struct {
	PathProp
	SizeVal int64 `json:"size,omitempty" yaml:"size,omitempty"`
	TimeVal int64 `json:"time,omitempty" yaml:"time,omitempty"`
}

// Setup fills fields from os.FileInfo structure. Do not looks for share.
func (fp *FileProp) Setup(fi os.FileInfo) {
	fp.NameVal = fi.Name()
	fp.SizeVal = fi.Size()
	fp.TimeVal = UnixJS(fi.ModTime())
}

// Size is file size in bytes.
func (fp *FileProp) Size() int64 {
	return fp.SizeVal
}

// Time is file creation time in UNIX format, milliseconds.
func (fp *FileProp) Time() int64 {
	return fp.TimeVal
}

// FileKit is common files properties kit.
type FileKit struct {
	FileProp
	TmbProp
}

// Setup calls nested structures setups.
func (fk *FileKit) Setup(syspath string, fi os.FileInfo) {
	fk.FileProp.Setup(fi)
	fk.TmbProp.Setup(syspath)
}

// FileGrp is files group alias.
type FileGrp [FGnum]int

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (fg *FileGrp) IsZero() bool {
	for _, v := range fg {
		if v != 0 {
			return false
		}
	}
	return true
}

// PathBase returns safe base of path or CID as is.
func PathBase(syspath string) string {
	if len(syspath) > 0 {
		var pos1 int
		var pos2 = len(syspath)
		if syspath[0] == '[' && syspath[pos2-1] == ']' {
			return syspath
		}
		if syspath[pos2-1] == '/' || syspath[pos2-1] == '\\' {
			pos2--
		}
		for pos1 = pos2; pos1 > 0 && syspath[pos1-1] != '/' && syspath[pos1-1] != '\\'; pos1-- {
		}
		return syspath[pos1:pos2]
	}
	return syspath
}

// DirProp is directory properties chunk.
type DirProp struct {
	// Directory scanning time in UNIX format, milliseconds.
	Scan int64 `json:"scan" yaml:"scan"`
	// Directory file groups counters.
	FGrp FileGrp `json:"fgrp" yaml:"fgrp,flow"`
}

// DirKit is directory properties kit.
type DirKit struct {
	PathProp
	TmbProp
	DirProp
}

// Setup fills fields with given path. Do not looks for share.
func (dk *DirKit) Setup(syspath string) {
	dk.NameVal = PathBase(syspath)
	dk.TypeVal = FTdir
	dk.PUIDVal = pathcache.Cache(syspath)
	dk.NTmbVal = TMBreject
	if dp, ok := dircache.Get(dk.PUIDVal); ok {
		dk.DirProp = dp
	}
}

// DriveKit is drive properties kit.
type DriveKit struct {
	PathProp
	TmbProp
	Latency int `json:"latency"` // drive connection latency in ms, or -1 on error
}

// Setup fills fields with given path. Do not looks for share.
func (dk *DriveKit) Setup(syspath string) {
	dk.NameVal = PathBase(syspath)
	dk.TypeVal = FTdrv
	dk.PUIDVal = pathcache.Cache(syspath)
	dk.NTmbVal = TMBreject
}

// Scan drive to check its latency.
func (dk *DriveKit) Scan(syspath string) error {
	var t1 = time.Now()
	var fi, err = os.Stat(syspath)
	if err == nil && !fi.IsDir() {
		err = ErrNotDir
	}
	if err == nil {
		dk.Latency = int(t1.Sub(time.Now()) / 1000000)
	} else {
		dk.Latency = -1
	}
	return err
}

// CatKit is categories properties kit.
type CatKit struct {
	PathProp
	TmbProp
	CID string `json:"cid"`
}

// Setup fills fields with given path. Do not looks for share.
func (ck *CatKit) Setup(fpath string) {
	var pos = strings.IndexByte(fpath, '/')
	ck.NameVal = fpath[pos+1 : len(fpath)-1]
	ck.TypeVal = FTctgr
	ck.PUIDVal = pathcache.Cache(fpath)
	ck.NTmbVal = TMBreject
	ck.CID = fpath[1:pos]
}

// TagEnum is descriptor for discs and tracks.
type TagEnum struct {
	Number int `json:"number,omitempty" yaml:"number,omitempty"`
	Total  int `json:"total,omitempty" yaml:"total,omitempty"`
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (te *TagEnum) IsZero() bool {
	return te.Number == 0 && te.Total == 0
}

// TagProp is Music file tags properties chunk.
type TagProp struct {
	Title    string  `json:"title,omitempty" yaml:"title,omitempty"`
	Album    string  `json:"album,omitempty" yaml:"album,omitempty"`
	Artist   string  `json:"artist,omitempty" yaml:"artist,omitempty"`
	Composer string  `json:"composer,omitempty" yaml:"composer,omitempty"`
	Genre    string  `json:"genre,omitempty" yaml:"genre,omitempty"`
	Year     int     `json:"year,omitempty" yaml:"year,omitempty"`
	Track    TagEnum `json:"track,omitempty" yaml:"track,flow,omitempty"`
	Disc     TagEnum `json:"disc,omitempty" yaml:"disc,flow,omitempty"`
	Lyrics   string  `json:"lyrics,omitempty" yaml:"lyrics,omitempty"`
	Comment  string  `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// Setup fills fields from tags metadata.
func (tp *TagProp) Setup(m tag.Metadata) {
	tp.Title = m.Title()
	tp.Album = m.Album()
	tp.Artist = m.Artist()
	tp.Composer = m.Composer()
	tp.Genre = m.Genre()
	tp.Year = m.Year()
	tp.Track.Number, tp.Track.Total = m.Track()
	tp.Disc.Number, tp.Disc.Total = m.Disc()
	tp.Lyrics = m.Lyrics()
	tp.Comment = m.Comment()
}

// TagKit is music file tags properties kit.
type TagKit struct {
	FileProp
	TmbProp
	TagProp
}

// Setup fills fields with given path.
// Puts into the cache nested at the tags thumbnail if it present.
func (tk *TagKit) Setup(syspath string, fi os.FileInfo) {
	tk.FileProp.Setup(fi)

	var md *MediaData
	if file, err := OpenFile(syspath); err == nil {
		defer file.Close()
		if m, err := tag.ReadFrom(file); err == nil {
			tk.TagProp.Setup(m)
			if pic := m.Picture(); pic != nil {
				if cfg.FitEmbeddedTmb {
					if md, err = MakeTmb(bytes.NewReader(pic.Data)); err != nil {
						md = &MediaData{
							Data: pic.Data,
							Mime: pic.MIMEType,
						}
					}
				} else {
					md = &MediaData{
						Data: pic.Data,
						Mime: pic.MIMEType,
					}
				}
			}
		}
	}
	tk.PUIDVal = pathcache.Cache(syspath)
	if md != nil {
		tk.NTmbVal = TMBcached
		thumbcache.Set(tk.PUIDVal, md)
	} else {
		tk.NTmbVal = TMBreject
	}
}

// GetTagTmb extracts embedded thumbnail from image file.
func GetTagTmb(syspath string) (md *MediaData, err error) {
	var file io.ReadSeekCloser
	if file, err = OpenFile(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err == nil {
		if pic := m.Picture(); pic != nil {
			if cfg.FitEmbeddedTmb {
				if md, err = MakeTmb(bytes.NewReader(pic.Data)); err != nil {
					md = &MediaData{
						Data: pic.Data,
						Mime: pic.MIMEType,
					}
				}
			} else {
				md = &MediaData{
					Data: pic.Data,
					Mime: pic.MIMEType,
				}
			}
		} else {
			err = ErrNotThumb
		}
	}
	return
}

// MakeProp is file properties factory.
func MakeProp(syspath string, fi os.FileInfo) Pather {
	if fi.IsDir() {
		var dk DirKit
		dk.Setup(syspath)
		return &dk
	}
	var ft = GetFileType(syspath)
	if ft == FTflac || ft == FTmus || ft == FTmp4 {
		var tk TagKit
		tk.TypeVal = ft
		tk.Setup(syspath, fi)
		return &tk
	} else if ft == FTjpeg || ft == FTtiff || ft == FTpng || ft == FTwebp {
		var ek ExifKit
		ek.TypeVal = ft
		ek.Setup(syspath, fi)
		return &ek
	} else {
		var fk FileKit
		fk.TypeVal = ft
		fk.Setup(syspath, fi)
		return &fk
	}
}

// The End.
