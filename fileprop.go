package hms

import (
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// FT_t is enum type for properties file types.
type FT_t int

// File types.
const (
	FTfile FT_t = 0
	FTdir  FT_t = 1
	FTdrv  FT_t = 2
	FTctgr FT_t = 3
)

// FG_t is enum type for file groups.
type FG_t int

// File groups.
const (
	FGother FG_t = 0
	FGvideo FG_t = 1
	FGaudio FG_t = 2
	FGimage FG_t = 3
	FGbooks FG_t = 4
	FGtexts FG_t = 5
	FGpacks FG_t = 6
	FGdir   FG_t = 7
)

// FGnum is count of file groups.
const FGnum = 8

var extgrp = map[string]FG_t{
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
	".avif": FGimage,
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

// GetFileExt returns file extension converted to lowercase.
func GetFileExt(fname string) string {
	return strings.ToLower(path.Ext(fname))
}

// GetFileGroup returns file group integer value for given file name by it's extension.
func GetFileGroup(fpath string) FG_t {
	return extgrp[GetFileExt(fpath)]
}

// IsTypeImage checks that file is some image format.
func IsTypeImage(ext string) bool {
	switch ext {
	case ".tga", ".bmp", ".dib", ".rle", ".dds", ".tif", ".tiff",
		".jpg", ".jpe", ".jpeg", ".jfif", ".gif", ".png", ".avif", ".webp",
		".psd", ".psb", ".jp2", ".jpg2", ".jpx", ".jpm", ".jxr":
		return true
	}
	return false
}

// IsTypeNativeImg checks that image file is supported by any browser without format conversion.
func IsTypeNativeImg(ext string) bool {
	switch ext {
	case ".jpg", ".jpe", ".jpeg", ".jfif",
		".avif", ".webp", ".png", ".gif":
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
	case ".avif", ".webp", ".png", ".gif",
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

// IsTypePlaylist checks that file extension belongs playlist file.
func IsTypePlaylist(ext string) bool {
	switch ext {
	case ".m3u", ".m3u8", ".wpl", ".pls", ".asx", ".xspf":
		return true
	}
	return false
}

// Pather is path properties interface.
type Pather interface {
	Name() string // string identifier
	Type() FT_t   // type identifier
	Size() int64  // size in bytes
	Time() Unix_t // UNIX time in milliseconds
}

// PathProp is any path base properties.
type PathProp struct {
	NameVal string `json:"name" yaml:"name" xml:"name"`
	TypeVal FT_t   `json:"type" yaml:"type" xml:"type"` // do not omit empty
}

// Name is file name with extension without path.
func (pp *PathProp) Name() string {
	return pp.NameVal
}

// Type is enumerated file type.
func (pp *PathProp) Type() FT_t {
	return pp.TypeVal
}

// Size is file size in bytes.
func (pp *PathProp) Size() int64 {
	return 0
}

// Time is file creation time in UNIX format, milliseconds.
func (pp *PathProp) Time() Unix_t {
	return 0
}

// FileProp is common file properties chunk.
type FileProp struct {
	PathProp `xorm:"extends" yaml:",inline"`
	SizeVal  int64  `xorm:"default 0" json:"size,omitempty" yaml:"size,omitempty" xml:"size,omitempty"`
	TimeVal  Unix_t `xorm:"default 0" json:"time,omitempty" yaml:"time,omitempty" xml:"time,omitempty"`
}

// Setup fills fields from fs.FileInfo structure. Do not looks for share.
func (fp *FileProp) Setup(fi fs.FileInfo) {
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
func (fp *FileProp) Time() Unix_t {
	return fp.TimeVal
}

// FileKit is common files properties kit.
type FileKit struct {
	FileProp `yaml:",inline"`
	PuidProp `yaml:",inline"`
	TmbProp  `yaml:",inline"`
}

// Setup calls nested structures setups.
func (fk *FileKit) Setup(syspath string, fi fs.FileInfo) {
	fk.FileProp.Setup(fi)
	fk.PuidProp.Setup(syspath)
	fk.TmbProp.Setup(syspath)
}

type FileGroup struct {
	FGother uint `xorm:"'other' default 0" json:"other,omitempty" yaml:"other,omitempty" xml,omitempty,attr:"other"`
	FGvideo uint `xorm:"'video' default 0" json:"video,omitempty" yaml:"video,omitempty" xml,omitempty,attr:"video"`
	FGaudio uint `xorm:"'audio' default 0" json:"audio,omitempty" yaml:"audio,omitempty" xml,omitempty,attr:"audio"`
	FGimage uint `xorm:"'image' default 0" json:"image,omitempty" yaml:"image,omitempty" xml,omitempty,attr:"image"`
	FGbooks uint `xorm:"'books' default 0" json:"books,omitempty" yaml:"books,omitempty" xml,omitempty,attr:"books"`
	FGtexts uint `xorm:"'texts' default 0" json:"texts,omitempty" yaml:"texts,omitempty" xml,omitempty,attr:"texts"`
	FGpacks uint `xorm:"'packs' default 0" json:"packs,omitempty" yaml:"packs,omitempty" xml,omitempty,attr:"packs"`
	FGdir   uint `xorm:"'dir' default 0" json:"dir,omitempty" yaml:"dir,omitempty" xml,omitempty,attr:"dir"`
}

// Field returns pointer to field value with given identifier.
func (fg *FileGroup) Field(id FG_t) *uint {
	switch id {
	case FGother:
		return &fg.FGother
	case FGvideo:
		return &fg.FGvideo
	case FGaudio:
		return &fg.FGaudio
	case FGimage:
		return &fg.FGimage
	case FGbooks:
		return &fg.FGbooks
	case FGtexts:
		return &fg.FGtexts
	case FGpacks:
		return &fg.FGpacks
	case FGdir:
		return &fg.FGdir
	default:
		return nil
	}
}

// Sum returns sum of all fields.
func (fg *FileGroup) Sum() uint {
	return fg.FGother + fg.FGvideo + fg.FGaudio + fg.FGimage + fg.FGbooks + fg.FGtexts + fg.FGpacks + fg.FGdir
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (fg *FileGroup) IsZero() bool {
	return fg.Sum() == 0
}

// PathBase returns safe base of path or CID as is.
func PathBase(syspath string) string {
	var pos1 int
	var pos2 = len(syspath)
	if pos2 == 0 {
		return ""
	}
	if syspath[0] == '<' && syspath[pos2-1] == '>' {
		return syspath
	}
	if syspath[pos2-1] == '/' || syspath[pos2-1] == '\\' {
		pos2--
	}
	for pos1 = pos2; pos1 > 0 && syspath[pos1-1] != '/' && syspath[pos1-1] != '\\'; pos1-- {
	}
	return syspath[pos1:pos2]
}

// DirProp is directory properties chunk.
type DirProp struct {
	Scan    Unix_t    `json:"scan,omitempty" yaml:"scan,omitempty" xml:"scan,omitempty"`                           // directory scanning time in UNIX format, milliseconds.
	FGrp    FileGroup `xorm:"extends" json:"fgrp,omitempty" yaml:"fgrp,flow,omitempty" xml:"fgrp,omitempty"`       // directory file groups counters.
	Latency int       `xorm:"default 0" json:"latency,omitempty" yaml:"latency,omitempty" xml:"latency,omitempty"` // drive connection latency in ms, or -1 on error
}

// DirKit is directory properties kit.
type DirKit struct {
	PathProp `yaml:",inline"`
	PuidProp `yaml:",inline"`
	DirProp  `yaml:",inline"`
}

// Setup fills fields with given path. Do not looks for share.
func (dk *DirKit) Setup(syspath string) {
	dk.NameVal = PathBase(syspath)
	dk.TypeVal = FTdir
	dk.PuidProp.Setup(syspath)
	if dp, ok := DirCacheGet(dk.PUIDVal); ok {
		dk.DirProp = dp
	}
}

// DriveKit is drive properties kit.
type DriveKit struct {
	PathProp `yaml:",inline"`
	PuidProp `yaml:",inline"`
	DirProp  `yaml:",inline"`
}

// Setup fills fields with given path. Do not looks for share.
func (dk *DriveKit) Setup(syspath string) {
	dk.NameVal = PathBase(syspath)
	dk.TypeVal = FTdrv
	dk.PuidProp.Setup(syspath)
	if dp, ok := DirCacheGet(dk.PUIDVal); ok {
		dk.DirProp = dp
	}
}

// Scan drive to check its latency.
func (dk *DriveKit) Scan(syspath string) error {
	var t1 = time.Now()
	var fi, err = os.Stat(syspath)
	if err == nil && !fi.IsDir() {
		err = ErrNotDir
	}
	if err == nil {
		dk.Latency = int(time.Since(t1) / time.Millisecond)
	} else {
		dk.Latency = -1
	}
	return err
}

// CatKit is category properties kit.
type CatKit struct {
	PathProp `yaml:",inline"`
	PuidProp `yaml:",inline"`
}

// Setup fills fields with given path. Do not looks for share.
func (ck *CatKit) Setup(puid Puid_t) {
	ck.NameVal = CatNames[puid]
	ck.TypeVal = FTctgr
	ck.PUIDVal = puid
}

// MakeProp is file properties factory.
func MakeProp(syspath string, fi fs.FileInfo) Pather {
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
