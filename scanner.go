package hms

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dhowden/tag"
)

// File types
const (
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
	FT_gif   = 10
	FT_png   = 11
	FT_jpeg  = 12
	FT_tiff  = 13
	FT_webp  = 14
	FT_pdf   = 15
	FT_html  = 16
	FT_text  = 17
	FT_scr   = 18
	FT_cfg   = 19
	FT_log   = 20
	FT_cab   = 21
	FT_zip   = 22
	FT_rar   = 23
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
	FT_gif:   FG_image,
	FT_png:   FG_image,
	FT_jpeg:  FG_image,
	FT_tiff:  FG_image,
	FT_webp:  FG_image,
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
	".gif":  FT_gif,
	".png":  FT_png,
	".jpg":  FT_jpeg,
	".jpe":  FT_jpeg,
	".jpeg": FT_jpeg,
	".jfif": FT_jpeg,
	".tif":  FT_tiff,
	".tiff": FT_tiff,
	".webp": FT_webp,

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

var (
	PhotoJPEG int64 = 2097152 // 2M
	PhotoWEBP int64 = 1572864 // 1.5M
)

const shareprefix = "/share/"

var shareslist = []FileProper{}      // plain list of active shares
var sharespath = map[string]string{} // active shares by full path
var sharespref = map[string]string{} // active shares by prefix
var sharesgone = map[string]string{} // gone shares by prefix
var shrmux sync.RWMutex

var root = DirProp{
	FileProp: FileProp{
		TypeVal: FT_dir,
		KTmbVal: ThumbName(""),
	},
}

var dircache = map[string]*DirProp{
	"/": &root,
}

var dcmux sync.RWMutex

// File properties interface.
type FileProper interface {
	Name() string
	Path() string
	Size() int64
	Time() int64
	Type() int
	Pref() string
	KTmb() string
	NTmb() int
	SetPref(string)
	SetNTmb(int)
}

// Common file properties.
type FileProp struct {
	NameVal string `json:"name,omitempty"`
	PathVal string `json:"path,omitempty"`
	SizeVal int64  `json:"size,omitempty"`
	TimeVal int64  `json:"time,omitempty"`
	TypeVal int    `json:"type,omitempty"`
	PrefVal string `json:"pref,omitempty"`
	KTmbVal string `json:"ktmb,omitempty"`
	NTmbVal int    `json:"ntmb,omitempty"`
}

// File name with extension without path.
func (fp *FileProp) Name() string {
	return fp.NameVal
}

// Full path with name; folder ends with splash.
func (fp *FileProp) Path() string {
	return fp.PathVal
}

// File size in bytes.
func (fp *FileProp) Size() int64 {
	return fp.SizeVal
}

// File creation time in UNIX format, milliseconds.
func (fp *FileProp) Time() int64 {
	return fp.TimeVal
}

// Enumerated file type.
func (fp *FileProp) Type() int {
	return fp.TypeVal
}

// Share prefix.
func (fp *FileProp) Pref() string {
	return fp.PrefVal
}

// Thumbnail key, it's MD5-hash of full path.
func (fp *FileProp) KTmb() string {
	return fp.KTmbVal
}

// Thumbnail state, -1 impossible, 0 undefined, 1 ready.
func (fp *FileProp) NTmb() int {
	return fp.NTmbVal
}

// Sets new share prefix value.
func (fp *FileProp) SetPref(pref string) {
	fp.PrefVal = pref
}

// Updates thumbnail state to given value.
func (fp *FileProp) SetNTmb(v int) {
	fp.NTmbVal = v
}

// Fills fields from os.FileInfo structure. Do not looks for share.
func (fp *FileProp) Setup(fi os.FileInfo, fpath string) {
	var fname, size = fi.Name(), fi.Size()
	var ktmb = ThumbName(fpath)
	fp.NameVal = fname
	fp.PathVal = fpath
	fp.SizeVal = size
	fp.TimeVal = UnixJS(fi.ModTime())
	fp.KTmbVal = ktmb
	if fi.IsDir() {
		fp.TypeVal = FT_dir
		fp.NTmbVal = TMB_reject
	} else {
		var ft = extset[strings.ToLower(filepath.Ext(fname))]
		if (ft == FT_jpeg && size > PhotoJPEG) || (ft == FT_webp && size > PhotoWEBP) {
			ft = FT_photo
		}
		fp.TypeVal = ft
		if tmb, err := thumbcache.Get(ktmb); err == nil {
			if tmb != nil {
				fp.NTmbVal = TMB_cached
			} else {
				fp.NTmbVal = TMB_reject
			}
		} else {
			fp.NTmbVal = TMB_none
		}
	}
}

// Directory properties.
type DirProp struct {
	FileProp
	// Directory scanning time in UNIX format, milliseconds.
	Scan int64 `json:"scan"`
	// Directory file groups counters.
	FGrp [FG_num]int `json:"fgrp"`
}

// Fills fields with given path. Do not looks for share.
func (dp *DirProp) Setup(fname, fpath string) {
	dp.NameVal = fname
	dp.PathVal = fpath
	dp.TypeVal = FT_dir
	dp.KTmbVal = ThumbName(fpath)
	dp.NTmbVal = TMB_reject
}

// Creates new DirProp or returns cached object with given full path.
func MakeDirProp(fpath string) *DirProp {
	dcmux.RLock()
	var dc, has = dircache[fpath]
	dcmux.RUnlock()
	if has {
		return dc
	}

	var _, fname = path.Split(fpath[:len(fpath)-1])
	var dp DirProp
	dp.Setup(fname, fpath)

	dcmux.Lock()
	dircache[fpath] = &dp
	dcmux.Unlock()

	return &dp
}

// Descriptor for discs and tracks.
type TagEnum struct {
	Number int `json:"number,omitempty"`
	Total  int `json:"total,omitempty"`
}

// Music file properties by file tags.
type TagProp struct {
	FileProp
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

// Fills fields with given path. Do not looks for share.
func (tp *TagProp) Setup(fi os.FileInfo, fpath string) (err error) {
	tp.FileProp.Setup(fi, fpath)

	var file *os.File
	if file, err = os.Open(fpath); err != nil {
		return err
	}
	defer file.Close()

	var m tag.Metadata
	if m, err = tag.ReadFrom(file); err != nil {
		return err
	}

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

	var pic = m.Picture()
	if pic != nil {
		tp.SetNTmb(TMB_cached)
		thumbcache.Set(tp.KTmbVal, &Thumb{
			Data: pic.Data,
			Mime: pic.MIMEType,
		})
	}

	return
}

// Returns os.FileInfo for given full file name.
func FileStat(fpath string) (fi os.FileInfo, err error) {
	var file *os.File
	file, err = os.Open(fpath)
	if err != nil {
		return
	}
	defer file.Close()
	fi, err = file.Stat()
	return
}

// File properties factory.
func MakeProp(fi os.FileInfo, fpath string) (prop FileProper) {
	if fi.IsDir() {
		prop = MakeDirProp(fpath)
	} else {
		var ft = extset[strings.ToLower(filepath.Ext(fpath))]
		if ft == FT_flac || ft == FT_mp3 || ft == FT_ogg || ft == FT_mp4 {
			var tp TagProp
			tp.Setup(fi, fpath)
			prop = &tp
		} else {
			var fp FileProp
			fp.Setup(fi, fpath)
			prop = &fp
		}
	}

	shrmux.RLock()
	if pref, ok := sharespath[fpath]; ok {
		prop.SetPref(pref)
	}
	shrmux.RUnlock()
	return
}

// Looks for correct prefix and add share with it.
func MakeShare(prop FileProper) {
	var pref string
	var name = prop.Name()
	if len(name) > 8 {
		pref = name[:8]
	} else {
		pref = name
	}
	var fit = true
	for _, b := range pref {
		if (b < '0' || b > '9') && (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && b != '-' && b != '_' {
			fit = false
		}
	}

	if fit && AddShare(pref, prop) {
		return
	}
	for i := 0; !AddShare(makerandstr(4), prop); i++ {
		if i > 1000 {
			panic("can not generate share prefix")
		}
	}
}

// Add share with given prefix.
func AddShare(pref string, prop FileProper) bool {
	shrmux.RLock()
	var _, ok = sharespref[pref]
	shrmux.RUnlock()

	if !ok {
		prop.SetPref(pref)
		var path = prop.Path()

		shrmux.Lock()
		shareslist = append(shareslist, prop)
		sharespath[path] = pref
		sharespref[pref] = path
		shrmux.Unlock()
	}
	return !ok
}

// Delete share by given prefix.
func DelSharePref(pref string) bool {
	shrmux.RLock()
	var path, ok = sharespref[pref]
	shrmux.RUnlock()

	if ok {
		shrmux.Lock()
		for i, fp := range shareslist {
			if fp.Pref() == pref {
				shareslist = append(shareslist[:i], shareslist[i+1:]...)
				break
			}
		}
		delete(sharespath, path)
		delete(sharespref, pref)
		sharesgone[pref] = path
		shrmux.Unlock()
	}
	return ok
}

// Delete share by given shared path.
func DelSharePath(path string) bool {
	shrmux.RLock()
	var pref, ok = sharespath[path]
	shrmux.RUnlock()

	if ok {
		shrmux.Lock()
		for i, fp := range shareslist {
			if fp.Path() == path {
				shareslist = append(shareslist[:i], shareslist[i+1:]...)
				break
			}
		}
		delete(sharespath, path)
		delete(sharespref, pref)
		sharesgone[pref] = path
		shrmux.Unlock()
	}
	return ok
}

// Returned data for "getdrv", "folder" API handlers.
type folderRet struct {
	Paths []*DirProp   `json:"paths"`
	Files []FileProper `json:"files"`
}

func (fr *folderRet) AddProp(prop FileProper) {
	if dp, ok := prop.(*DirProp); ok {
		fr.Paths = append(fr.Paths, dp)
	} else {
		fr.Files = append(fr.Files, prop)
	}
}

var sharecharset = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func makerandstr(n int) string {
	var l = byte(len(sharecharset))
	var str = make([]byte, n)
	randbytes(str)
	for i := 0; i < n; i++ {
		str[i] = sharecharset[str[i]%l]
	}
	return string(str)
}

// Scan all available drives installed on local machine.
func getdrives() (drvs []*DirProp) {
	for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		var fname = string(drive) + ":"
		var fpath = fname + "/"
		var _, err = FileStat(fpath)
		if err != nil {
			continue
		}
		var dp = MakeDirProp(fpath)
		dp.NameVal = fname
		drvs = append(drvs, dp)
	}

	root.Scan = UnixJSNow()
	root.FGrp[FG_dir] = len(drvs)
	return
}

// Reads directory with given name and returns FileProper for each entry.
func readdir(dirname string) (ret folderRet, err error) {
	defer func() {
		// Remove from cache dir that can not be opened
		if err != nil {
			dcmux.Lock()
			delete(dircache, dirname)
			dcmux.Unlock()
		}
	}()

	if !strings.HasSuffix(dirname, "/") {
		dirname += "/"
	}

	var file *os.File
	if file, err = os.Open(dirname); err != nil {
		return
	}
	var fis []os.FileInfo
	fis, err = file.Readdir(-1)
	file.Close()
	if err != nil {
		return
	}

	var fgrp = [FG_num]int{}

scanprop:
	for _, fi := range fis {
		var fname = fi.Name()
		for _, pat := range hidden {
			if matched, _ := path.Match(pat, strings.ToLower(fname)); matched {
				continue scanprop
			}
		}

		var fpath = dirname + fname
		if fi.IsDir() {
			fpath += "/"
		}

		var prop = MakeProp(fi, fpath)
		fgrp[typetogroup[prop.Type()]]++
		ret.AddProp(prop)
	}

	var dp = MakeDirProp(dirname)
	dp.Scan = UnixJSNow()
	dp.FGrp = fgrp

	return
}

// The End.
