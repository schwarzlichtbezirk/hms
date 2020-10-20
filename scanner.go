package hms

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// File types
const (
	FT_drive = -2
	FT_dir   = -1
	FT_file  = 0
	FT_wave  = 1
	FT_flac  = 2
	FT_mp3   = 3
	FT_ogg   = 4
	FT_mp4   = 5
	FT_webm  = 6
	FT_photo = 7
	FT_tga   = 8
	FT_bmp   = 9
	FT_dds   = 10
	FT_tiff  = 11
	FT_jpeg  = 12
	FT_gif   = 13
	FT_png   = 14
	FT_webp  = 15
	FT_psd   = 16
	FT_pdf   = 17
	FT_html  = 18
	FT_text  = 19
	FT_scr   = 20
	FT_cfg   = 21
	FT_log   = 22
	FT_cab   = 23
	FT_zip   = 24
	FT_rar   = 25
)

// File groups
const (
	FG_other = 0
	FG_music = 1
	FG_video = 2
	FG_image = 3
	FG_books = 4
	FG_texts = 5
	FG_store = 6
	FG_dir   = 7
)

const FG_num = 8

var typetogroup = map[int]int{
	FT_drive: FG_dir,
	FT_dir:   FG_dir,
	FT_file:  FG_other,
	FT_wave:  FG_music,
	FT_flac:  FG_music,
	FT_mp3:   FG_music,
	FT_ogg:   FG_music,
	FT_mp4:   FG_video,
	FT_webm:  FG_video,
	FT_photo: FG_image,
	FT_tga:   FG_image,
	FT_bmp:   FG_image,
	FT_dds:   FG_image,
	FT_tiff:  FG_image,
	FT_jpeg:  FG_image,
	FT_gif:   FG_image,
	FT_png:   FG_image,
	FT_webp:  FG_image,
	FT_psd:   FG_image,
	FT_pdf:   FG_books,
	FT_html:  FG_books,
	FT_text:  FG_texts,
	FT_scr:   FG_texts,
	FT_cfg:   FG_texts,
	FT_log:   FG_texts,
	FT_cab:   FG_store,
	FT_zip:   FG_store,
	FT_rar:   FG_store,
}

var extset = map[string]int{
	// Audio
	".wav":  FT_wave,
	".flac": FT_flac,
	".mp3":  FT_mp3,
	".ogg":  FT_ogg,

	// Video
	".mp4":  FT_mp4,
	".webm": FT_webm,

	// Images
	".tga":  FT_tga,
	".bmp":  FT_bmp,
	".dib":  FT_bmp,
	".dds":  FT_dds,
	".tif":  FT_tiff,
	".tiff": FT_tiff,
	".jpg":  FT_jpeg,
	".jpe":  FT_jpeg,
	".jpeg": FT_jpeg,
	".jfif": FT_jpeg,
	".gif":  FT_gif,
	".png":  FT_png,
	".webp": FT_webp,
	".psd":  FT_psd,
	".psb":  FT_psd,

	// Text
	".pdf":   FT_pdf,
	".html":  FT_html,
	".htm":   FT_html,
	".shtml": FT_html,
	".shtm":  FT_html,
	".xhtml": FT_html,
	".phtml": FT_html,
	".hta":   FT_html,
	".txt":   FT_text,
	".css":   FT_scr,
	".js":    FT_scr,
	".jsm":   FT_scr,
	".vb":    FT_scr,
	".vbs":   FT_scr,
	".bat":   FT_scr,
	".cmd":   FT_scr,
	".sh":    FT_scr,
	".mak":   FT_scr,
	".iss":   FT_scr,
	".nsi":   FT_scr,
	".nsh":   FT_scr,
	".bsh":   FT_scr,
	".sql":   FT_scr,
	".as":    FT_scr,
	".mx":    FT_scr,
	".php":   FT_scr,
	".phpt":  FT_scr,
	".java":  FT_scr,
	".jsp":   FT_scr,
	".asp":   FT_scr,
	".lua":   FT_scr,
	".tcl":   FT_scr,
	".asm":   FT_scr,
	".c":     FT_scr,
	".h":     FT_scr,
	".hpp":   FT_scr,
	".hxx":   FT_scr,
	".cpp":   FT_scr,
	".cxx":   FT_scr,
	".cc":    FT_scr,
	".cs":    FT_scr,
	".go":    FT_scr,
	".r":     FT_scr,
	".d":     FT_scr,
	".pas":   FT_scr,
	".inc":   FT_scr,
	".py":    FT_scr,
	".pyw":   FT_scr,
	".pl":    FT_scr,
	".pm":    FT_scr,
	".plx":   FT_scr,
	".rb":    FT_scr,
	".rbw":   FT_scr,
	".rc":    FT_scr,
	".ps":    FT_scr,
	".ini":   FT_cfg,
	".inf":   FT_cfg,
	".reg":   FT_cfg,
	".url":   FT_cfg,
	".xml":   FT_cfg,
	".xsml":  FT_cfg,
	".xsl":   FT_cfg,
	".xsd":   FT_cfg,
	".kml":   FT_cfg,
	".wsdl":  FT_cfg,
	".xlf":   FT_cfg,
	".xliff": FT_cfg,
	".yml":   FT_cfg,
	".cmake": FT_cfg,
	".vhd":   FT_cfg,
	".vhdl":  FT_cfg,
	".json":  FT_cfg,
	".log":   FT_log,

	// archive
	".cab": FT_cab,
	".tar": FT_cab,
	".zip": FT_zip,
	".7z":  FT_zip,
	".rar": FT_rar,
}

// File properties interface.
type Proper interface {
	Name() string // string identifier
	Size() int64  // size in bytes
	Time() int64  // UNIX time in milliseconds
	Type() int    // type identifier
	Hash() string // MD5-hash of full file path encoded to hex-base32
	NTmb() int    // -1 - can not make thumbnail; 0 - not cached; 1 - cached
	SetNTmb(int)
	Clone() Proper
}

type ShareKit struct {
	Prop Proper `json:"prop"`
	Path string `json:"path,omitempty"` // full share path to file or directory
	Pref string `json:"pref,omitempty"` // share prefix
}

// Sets full path value.
func (sk *ShareKit) SetPath(path string) {
	sk.Path = path
	sk.Pref = ""
}

// Sets new share prefix value and updates full path.
func (sk *ShareKit) SetPref(pref string) {
	sk.Pref = pref
	sk.Path = pref
	if sk.Prop.Type() < 0 {
		sk.Path += "/"
	}
}

// Common file properties chunk.
type StdProp struct {
	NameVal string `json:"name,omitempty"`
	PathVal string `json:"path,omitempty"`
	SizeVal int64  `json:"size,omitempty"`
	TimeVal int64  `json:"time,omitempty"`
	TypeVal int    `json:"type,omitempty"`
	PrefVal string `json:"pref,omitempty"`
}

// Fills fields from os.FileInfo structure. Do not looks for share.
func (sp *StdProp) Setup(fi os.FileInfo) {
	sp.NameVal = fi.Name()
	sp.SizeVal = fi.Size()
	sp.TimeVal = UnixJS(fi.ModTime())
}

// File name with extension without path.
func (sp *StdProp) Name() string {
	return sp.NameVal
}

// File size in bytes.
func (sp *StdProp) Size() int64 {
	return sp.SizeVal
}

// File creation time in UNIX format, milliseconds.
func (sp *StdProp) Time() int64 {
	return sp.TimeVal
}

// Enumerated file type.
func (sp *StdProp) Type() int {
	return sp.TypeVal
}

// Common files properties kit.
type FileKit struct {
	StdProp
	TmbProp
}

// Creates copy of it self.
func (fk *FileKit) Clone() Proper {
	var c = *fk
	return &c
}

// Calls nested structures setups.
func (fk *FileKit) Setup(syspath string, fi os.FileInfo) {
	fk.StdProp.Setup(fi)
	fk.TmbProp.Setup(syspath)
}

// Directory properties chunk.
type DirProp struct {
	// Directory scanning time in UNIX format, milliseconds.
	Scan int64 `json:"scan"`
	// Directory file groups counters.
	FGrp [FG_num]int `json:"fgrp"`
}

// Directory properties kit.
type DirKit struct {
	StdProp
	TmbProp
	DirProp
}

// Creates copy of it self.
func (dk *DirKit) Clone() Proper {
	var c = *dk
	return &c
}

// Fills fields with given path. Do not looks for share.
func (dk *DirKit) Setup(syspath string) {
	dk.NameVal = filepath.Base(syspath)
	dk.TypeVal = FT_dir
	dk.HashVal = hashcache.Cache(syspath)
	dk.NTmbVal = TMB_reject
}

type DriveKit struct {
	DirKit
	Latency int `json:"latency"` // drive connection latency in ms, or -1 on error
}

// Creates copy of it self.
func (dk *DriveKit) Clone() Proper {
	var c = *dk
	return &c
}

// Fills fields with given path. Do not looks for share.
func (dk *DriveKit) Setup(syspath string) {
	dk.NameVal = syspath[:len(syspath)-1]
	dk.TypeVal = FT_drive
	dk.HashVal = hashcache.Cache(syspath)
	dk.NTmbVal = TMB_reject
}

func (dk *DriveKit) Scan(syspath string) error {
	var t1 = time.Now()
	var fi, err = os.Stat(syspath)
	if err == nil && !fi.IsDir() {
		err = ErrNotPath
	}
	if err == nil {
		dk.Latency = int(t1.Sub(time.Now()) / 1000000)
	} else {
		dk.Latency = -1
	}
	return err
}

// Descriptor for discs and tracks.
type TagEnum struct {
	Number int `json:"number,omitempty"`
	Total  int `json:"total,omitempty"`
}

// Music file tags properties chunk.
type TagProp struct {
	Title    string  `json:"title,omitempty"`
	Album    string  `json:"album,omitempty"`
	Artist   string  `json:"artist,omitempty"`
	Composer string  `json:"composer,omitempty"`
	Genre    string  `json:"genre,omitempty"`
	Year     int     `json:"year,omitempty"`
	Track    TagEnum `json:"track,omitempty"`
	Disc     TagEnum `json:"disc,omitempty"`
	Lyrics   string  `json:"lyrics,omitempty"`
	Comment  string  `json:"comment,omitempty"`
}

// Fills fields from tags metadata.
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

// Music file tags properties kit.
type TagKit struct {
	StdProp
	TmbProp
	TagProp
}

// Creates copy of it self.
func (tk *TagKit) Clone() Proper {
	var c = *tk
	return &c
}

// Fills fields with given path.
// Puts into the cache nested at the tags thumbnail if it present.
func (tk *TagKit) Setup(syspath string, fi os.FileInfo) {
	tk.StdProp.Setup(fi)

	if file, err := os.Open(syspath); err == nil {
		defer file.Close()
		if m, err := tag.ReadFrom(file); err == nil {
			tk.TagProp.Setup(m)
			if pic := m.Picture(); pic != nil {
				tk.HashVal = hashcache.Cache(syspath)
				tk.NTmbVal = TMB_cached
				thumbcache.Set(tk.HashVal, &ThumbElem{
					Data: pic.Data,
					Mime: pic.MIMEType,
				})
				return
			}
		}
	}
	tk.TmbProp.Setup(syspath)
}

func GetTagTmb(syspath string) (tmb *ThumbElem, err error) {
	var file *os.File
	if file, err = os.Open(syspath); err != nil {
		return // can not open file
	}
	defer file.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err == nil {
		if pic := m.Picture(); pic != nil {
			tmb = &ThumbElem{
				Data: pic.Data,
				Mime: pic.MIMEType,
			}
			return
		} else {
			err = ErrNotThumb
		}
	}
	return
}

// File properties factory.
func MakeProp(syspath string, fi os.FileInfo) Proper {
	if fi.IsDir() {
		var dk DirKit
		dk.Setup(syspath)
		return &dk
	} else {
		var ft = extset[strings.ToLower(filepath.Ext(syspath))]
		if ft == FT_flac || ft == FT_mp3 || ft == FT_ogg || ft == FT_mp4 {
			var tk TagKit
			tk.TypeVal = ft
			tk.Setup(syspath, fi)
			return &tk
		} else if ft == FT_jpeg || ft == FT_tiff || ft == FT_png || ft == FT_webp {
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
}

// File properties factory.
func CacheProp(syspath string, fi os.FileInfo) Proper {
	if propcache.Has(syspath) {
		var cp, _ = propcache.Get(syspath)
		return cp.(Proper)
	}
	var prop = MakeProp(syspath, fi)
	propcache.Set(syspath, prop)
	return prop
}

// The End.
