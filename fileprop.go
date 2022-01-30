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
	FTfile = 0
	FTdir  = 1
	FTdrv  = 2
	FTctgr = 3
)

// File groups
const (
	FGother = 0
	FGvideo = 1
	FGaudio = 2
	FGimage = 3
	FGbooks = 4
	FGtexts = 5
	FGpacks = 6
	FGdir   = 7
)

// FGnum is count of file groups
const FGnum = 8

var extgrp = map[string]int{
	// Video
	".avi":  FGvideo,
	".mpe":  FGvideo,
	".mpg":  FGvideo,
	".mp4":  FGvideo,
	".webm": FGvideo,
	".wmv":  FGvideo,
	".wmx":  FGvideo,
	".flv":  FGvideo,
	".3gp":  FGvideo,
	".3g2":  FGvideo,
	".mkv":  FGvideo,
	".mov":  FGvideo,
	".ogv":  FGvideo,
	".ogx":  FGvideo,

	// Audio
	".aac":  FGaudio,
	".m4a":  FGaudio,
	".alac": FGaudio,
	".aif":  FGaudio,
	".mpa":  FGaudio,
	".mp3":  FGaudio,
	".wav":  FGaudio,
	".wma":  FGaudio,
	".weba": FGaudio,
	".oga":  FGaudio,
	".ogg":  FGaudio,
	".opus": FGaudio,
	".flac": FGaudio,
	".mka":  FGaudio,
	".ra":   FGaudio,
	".mid":  FGaudio,
	".midi": FGaudio,
	".cda":  FGaudio,

	// Images
	".tga":  FGimage,
	".bmp":  FGimage,
	".dib":  FGimage,
	".rle":  FGimage,
	".dds":  FGimage,
	".tif":  FGimage,
	".tiff": FGimage,
	".jpg":  FGimage,
	".jpe":  FGimage,
	".jpeg": FGimage,
	".jfif": FGimage,
	".gif":  FGimage,
	".png":  FGimage,
	".webp": FGimage,
	".psd":  FGimage,
	".psb":  FGimage,
	".jp2":  FGimage,
	".jpg2": FGimage,
	".jpx":  FGimage,
	".jpm":  FGimage,
	".jxr":  FGimage,

	// Books
	".pdf":   FGbooks,
	".djvu":  FGbooks,
	".djv":   FGbooks,
	".html":  FGbooks,
	".htm":   FGbooks,
	".shtml": FGbooks,
	".shtm":  FGbooks,
	".xhtml": FGbooks,
	".phtml": FGbooks,
	".hta":   FGbooks,
	".mht":   FGbooks,
	// Office
	".odt":  FGbooks,
	".ods":  FGbooks,
	".odp":  FGbooks,
	".rtf":  FGbooks,
	".abw":  FGbooks,
	".doc":  FGbooks,
	".docx": FGbooks,
	".xls":  FGbooks,
	".xlsx": FGbooks,
	".ppt":  FGbooks,
	".pptx": FGbooks,
	".vsd":  FGbooks,

	// Texts
	".txt":   FGtexts,
	".md":    FGtexts,
	".css":   FGtexts,
	".js":    FGtexts,
	".jsm":   FGtexts,
	".vb":    FGtexts,
	".vbs":   FGtexts,
	".bat":   FGtexts,
	".cmd":   FGtexts,
	".sh":    FGtexts,
	".mak":   FGtexts,
	".iss":   FGtexts,
	".nsi":   FGtexts,
	".nsh":   FGtexts,
	".bsh":   FGtexts,
	".sql":   FGtexts,
	".as":    FGtexts,
	".mx":    FGtexts,
	".ps":    FGtexts,
	".php":   FGtexts,
	".phpt":  FGtexts,
	".lua":   FGtexts,
	".tcl":   FGtexts,
	".rc":    FGtexts,
	".cmake": FGtexts,
	".java":  FGtexts,
	".jsp":   FGtexts,
	".asp":   FGtexts,
	".asm":   FGtexts,
	".c":     FGtexts,
	".h":     FGtexts,
	".hpp":   FGtexts,
	".hxx":   FGtexts,
	".cpp":   FGtexts,
	".cxx":   FGtexts,
	".cc":    FGtexts,
	".cs":    FGtexts,
	".go":    FGtexts,
	".r":     FGtexts,
	".d":     FGtexts,
	".pas":   FGtexts,
	".inc":   FGtexts,
	".py":    FGtexts,
	".pyw":   FGtexts,
	".pl":    FGtexts,
	".pm":    FGtexts,
	".plx":   FGtexts,
	".rb":    FGtexts,
	".rbw":   FGtexts,
	".cfg":   FGtexts,
	".ini":   FGtexts,
	".inf":   FGtexts,
	".reg":   FGtexts,
	".url":   FGtexts,
	".xml":   FGtexts,
	".xsml":  FGtexts,
	".xsl":   FGtexts,
	".xsd":   FGtexts,
	".kml":   FGtexts,
	".gpx":   FGtexts,
	".wsdl":  FGtexts,
	".xlf":   FGtexts,
	".xliff": FGtexts,
	".yml":   FGtexts,
	".yaml":  FGtexts,
	".json":  FGtexts,
	".log":   FGtexts,

	// storage
	".cab":  FGpacks,
	".zip":  FGpacks,
	".7z":   FGpacks,
	".rar":  FGpacks,
	".rev":  FGpacks,
	".jar":  FGpacks,
	".apk":  FGpacks,
	".tar":  FGpacks,
	".tgz":  FGpacks,
	".gz":   FGpacks,
	".bz2":  FGpacks,
	".iso":  FGpacks,
	".isz":  FGpacks,
	".udf":  FGpacks,
	".nrg":  FGpacks,
	".mdf":  FGpacks,
	".mdx":  FGpacks,
	".img":  FGpacks,
	".ima":  FGpacks,
	".imz":  FGpacks,
	".ccd":  FGpacks,
	".vc4":  FGpacks,
	".dmg":  FGpacks,
	".daa":  FGpacks,
	".uif":  FGpacks,
	".vhd":  FGpacks,
	".vhdx": FGpacks,
	".vmdk": FGpacks,
	".wpk":  FGpacks,
	".m3u":  FGpacks,
	".m3u8": FGpacks,
	".wpl":  FGpacks,
	".pls":  FGpacks,
	".asx":  FGpacks,
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

// GetFileExt returns file extension converted to lowercase.
func GetFileExt(fname string) string {
	return strings.ToLower(path.Ext(fname))
}

// GetFileGroup returns file group integer value for given file name by it's extension.
func GetFileGroup(fpath string) int {
	return extgrp[GetFileExt(fpath)]
}

// IsTypeNativeImg checks that image file is supported by any browser without format conversion.
func IsTypeNativeImg(ext string) bool {
	switch ext {
	case ".jpg", ".jpe", ".jpeg", ".jfif",
		".png", ".webp", ".gif":
		return true
	}
	return false
}

// IsTypeJPEG checks that file extension is in JPEG group.
func IsTypeJPEG(ext string) bool {
	switch ext {
	case ".jpg", ".jpe", ".jpeg", ".jfif":
		return true
	}
	return false
}

// IsTypeAlpha checks that file extension belongs to images with alpha channel.
func IsTypeAlpha(ext string) bool {
	switch ext {
	case ".png", ".webp", ".gif",
		".dds", ".psd", ".psb":
		return true
	}
	return false
}

// IsTypeNonalpha checks that file extension belongs to images without alpha channel.
func IsTypeNonalpha(ext string) bool {
	switch ext {
	case ".jpg", ".jpe", ".jpeg", ".jfif",
		".tga", ".bmp", ".dib", ".rle", ".tif", ".tiff":
		return true
	}
	return false
}

// IsTypeID3 checks that file extension belongs to audio/video files with ID3 tags.
func IsTypeID3(ext string) bool {
	switch ext {
	case ".mp3", ".flac", ".ogg", ".opus", ".wma", ".mp4", ".acc", ".m4a", ".alac":
		return true
	}
	return false
}

// IsTypeEXIF checks that file extension belongs to images with EXIF tags.
func IsTypeEXIF(ext string) bool {
	switch ext {
	case ".tif", ".tiff",
		".jpg", ".jpe", ".jpeg", ".jfif",
		".png", ".webp":
		return true
	}
	return false
}

// Pather is path properties interface.
type Pather interface {
	Name() string   // string identifier
	Type() int      // type identifier
	Size() int64    // size in bytes
	Time() int64    // UNIX time in milliseconds
	PUID() PuidType // path unique ID encoded to hex-base32
	NTmb() int      // -1 - can not make thumbnail; 0 - not cached; 1 - cached
	MTmb() string   // thumbnail MIME type
	SetTmb(int, string)
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
	fp.TypeVal = FTfile
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
	dk.PUIDVal = syspathcache.Cache(syspath)
	dk.SetTmb(TMBreject, "")
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
	dk.PUIDVal = syspathcache.Cache(syspath)
	dk.SetTmb(TMBreject, "")
}

// Scan drive to check its latency.
func (dk *DriveKit) Scan(syspath string) error {
	var t1 = time.Now()
	var fi, err = os.Stat(syspath)
	if err == nil && !fi.IsDir() {
		err = ErrNotDir
	}
	if err == nil {
		dk.Latency = int(time.Until(t1) / 1000000)
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
	ck.PUIDVal = syspathcache.Cache(fpath)
	ck.SetTmb(TMBreject, "")
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
	if file, err := OpenFile(syspath); err == nil { // VFile
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
	tk.PUIDVal = syspathcache.Cache(syspath)
	if md != nil {
		tk.SetTmb(TMBcached, md.Mime)
		thumbcache.Set(tk.PUIDVal, md)
	} else {
		tk.SetTmb(TMBreject, "")
	}
}

// GetTagTmb extracts embedded thumbnail from image file.
func GetTagTmb(syspath string) (md *MediaData, err error) {
	var file io.ReadSeekCloser // VFile
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
	var ext = GetFileExt(syspath)
	if IsTypeID3(ext) {
		var tk TagKit
		tk.Setup(syspath, fi)
		return &tk
	} else if IsTypeEXIF(ext) {
		var ek ExifKit
		ek.Setup(syspath, fi)
		return &ek
	} else {
		var fk FileKit
		fk.Setup(syspath, fi)
		return &fk
	}
}

// The End.
