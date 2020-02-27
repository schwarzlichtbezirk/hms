package hms

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	FT_bmp   = 8
	FT_gif   = 9
	FT_png   = 10
	FT_jpeg  = 11
	FT_webp  = 12
	FT_pdf   = 13
	FT_html  = 14
	FT_text  = 15
	FT_scr   = 16
	FT_cfg   = 17
	FT_log   = 18
)

// File groups
const (
	FG_other = 0
	FG_music = 1
	FG_video = 2
	FG_image = 3
	FG_books = 4
	FG_texts = 5
	FG_dir   = 6
)

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
	FT_bmp:   FG_image,
	FT_gif:   FG_image,
	FT_png:   FG_image,
	FT_jpeg:  FG_image,
	FT_webp:  FG_image,
	FT_pdf:   FG_books,
	FT_html:  FG_books,
	FT_text:  FG_texts,
	FT_scr:   FG_texts,
	FT_cfg:   FG_texts,
	FT_log:   FG_texts,
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
	".bmp":  FT_bmp,
	".dib":  FT_bmp,
	".gif":  FT_gif,
	".png":  FT_png,
	".jpg":  FT_jpeg,
	".jpe":  FT_jpeg,
	".jpeg": FT_jpeg,
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
}

var (
	PhotoJPEG int64 = 2097152 // 2M
	PhotoWEBP int64 = 1572864 // 1.5M
)

const shareprefix = "/share/"

var shareslist []*FileProp              // plain list of active shares
var sharespath = map[string]*FileProp{} // active shares by full path
var sharespref = map[string]*FileProp{} // active shares by prefix
var sharesgone = map[string]*FileProp{} // gone shares by prefix
var shrmux sync.RWMutex

var root = DirProp{}
var dircache = map[string]*DirProp{
	"/": &root,
}
var dcmux sync.RWMutex

////////////////////////////
// Common file properties //
////////////////////////////

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

type FileProp struct {
	Name string `json:"name"`
	Path string `json:"path"` // full path with name; folder ends with splash
	Size int64  `json:"size,omitempty"`
	Time int64  `json:"time,omitempty"`
	Type int    `json:"type"`
	Pref string `json:"pref,omitempty"` // share prefix
	KTmb string `json:"ktmb,omitempty"` // thumbnail key
	NTmb int    `json:"ntmb"`           // thumbnail state, -1 impossible, 0 undefined, 1 ready
}

func (fp *FileProp) Setup(fi os.FileInfo, fpath string) {
	fp.Name = fi.Name()
	fp.Path = fpath
	fp.Size = fi.Size()
	fp.Time = UnixJS(fi.ModTime())
	fp.KTmb = ThumbName(fpath)
	if fi.IsDir() {
		fp.Type = FT_dir
		fp.NTmb = TMB_reject
	} else {
		fp.Type = extset[strings.ToLower(filepath.Ext(fp.Name))]
		if (fp.Type == FT_jpeg && fp.Size > PhotoJPEG) || (fp.Type == FT_webp && fp.Size > PhotoWEBP) {
			fp.Type = FT_photo
		}
		if tmb, err := thumbcache.Get(fp.KTmb); err == nil {
			if tmb != nil {
				fp.NTmb = TMB_cached
			} else {
				fp.NTmb = TMB_reject
			}
		} else {
			fp.NTmb = TMB_none
		}
	}
}

func (fp *FileProp) MakeShare() {
	var pref string
	if len(fp.Name) > 8 {
		pref = fp.Name[:8]
	} else {
		pref = fp.Name
	}
	var fit = true
	for _, b := range pref {
		if (b < '0' || b > '9') && (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && b != '-' && b != '_' {
			fit = false
		}
	}

	if fit && AddShare(pref, fp) {
		return
	}
	for i := 0; !AddShare(makerandstr(4), fp); i++ {
		if i > 1000 {
			panic("can not generate share prefix")
		}
	}
}

func AddShare(pref string, fp *FileProp) bool {
	shrmux.RLock()
	var _, ok = sharespref[pref]
	shrmux.RUnlock()

	if !ok {
		fp.Pref = pref

		shrmux.Lock()
		shareslist = append(shareslist, fp)
		sharespath[fp.Path] = fp
		sharespref[pref] = fp
		shrmux.Unlock()
	}
	return !ok
}

func DelSharePref(pref string) bool {
	shrmux.RLock()
	var shr, ok = sharespref[pref]
	shrmux.RUnlock()

	if ok {
		shrmux.Lock()
		for i, fp := range shareslist {
			if fp.Pref == pref {
				shareslist = append(shareslist[:i], shareslist[i+1:]...)
				break
			}
		}
		delete(sharespath, shr.Path)
		delete(sharespref, pref)
		sharesgone[pref] = shr
		shrmux.Unlock()
	}
	return ok
}

func DelSharePath(path string) bool {
	shrmux.RLock()
	var shr, ok = sharespath[path]
	shrmux.RUnlock()

	if ok {
		shrmux.Lock()
		for i, fp := range shareslist {
			if fp.Path == path {
				shareslist = append(shareslist[:i], shareslist[i+1:]...)
				break
			}
		}
		delete(sharespath, path)
		delete(sharespref, shr.Pref)
		sharesgone[shr.Pref] = shr
		shrmux.Unlock()
	}
	return ok
}

//////////////////////////
// Directory properties //
//////////////////////////

type DirProp struct {
	FileProp
	Scan int64  `json:"scan"` // scanning time
	FGrp [7]int `json:"fgrp"` // file groups counters
}

// Returned data for "getdrv", "folder" API handlers.
type folderRet struct {
	Paths []*DirProp  `json:"paths"`
	Files []*FileProp `json:"files"`
}

// Scan all available drives installed on local machine.
func getdrives() (drvs []*DirProp) {
	for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		var fname = string(drive) + ":"
		var fpath = fname + "/"
		var file, err = os.Open(fname)
		if err == nil {
			file.Close()
			var dp = DirProp{
				FileProp: FileProp{
					Name: fname,
					Path: fpath,
					Type: FT_dir,
					KTmb: ThumbName(fpath),
					NTmb: TMB_reject,
				},
			}
			shrmux.RLock()
			if shr, ok := sharespath[dp.Path]; ok {
				dp.Pref = shr.Pref
			}
			shrmux.RUnlock()
			drvs = append(drvs, &dp)
		}
	}

	root.Scan = UnixJS(time.Now())
	root.FGrp[FG_dir] = len(drvs)
	return
}

// Reads directory with given name and returns fileinfo for each entry.
func readdir(dirname string) (ret folderRet, err error) {
	defer func() {
		// Remove from cache dir that can not be opened
		if err != nil {
			dcmux.Lock()
			delete(dircache, dirname)
			dcmux.Unlock()
		}
	}()

	var last = dirname[len(dirname)-1]
	if last != '/' {
		dirname += "/"
	}

	var file *os.File
	file, err = os.Open(dirname)
	if err != nil {
		return
	}
	var fis []os.FileInfo
	fis, err = file.Readdir(-1)
	file.Close()
	if err != nil {
		return
	}
	var fgrp = [7]int{}
	var scan = UnixJS(time.Now())

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
		var fp FileProp
		fp.Setup(fi, fpath)

		fgrp[typetogroup[fp.Type]]++

		shrmux.RLock()
		if shr, ok := sharespath[fpath]; ok {
			fp.Pref = shr.Pref
		}
		shrmux.RUnlock()

		if fp.Type == FT_dir {
			dcmux.RLock()
			var dc, ok = dircache[fpath]
			dcmux.RUnlock()
			var dp = DirProp{
				FileProp: fp,
			}
			if ok {
				dp.Scan = dc.Scan
				dp.FGrp = dc.FGrp
			}
			ret.Paths = append(ret.Paths, &dp)
		} else {
			ret.Files = append(ret.Files, &fp)
		}
	}

	dcmux.RLock()
	var cached, cchok = dircache[dirname]
	dcmux.RUnlock()
	if cchok {
		cached.Scan = scan
		cached.FGrp = fgrp
	} else {
		var _, fname = path.Split(dirname[:len(dirname)-1])
		var dp = DirProp{
			FileProp: FileProp{
				Name: fname,
				Path: dirname,
				Type: FT_dir,
				KTmb: ThumbName(dirname),
				NTmb: TMB_reject,
			},
			Scan: scan,
			FGrp: fgrp,
		}

		dcmux.Lock()
		dircache[dirname] = &dp
		dcmux.Unlock()
	}

	return
}

// The End.
