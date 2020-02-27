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
	Dir    = -1
	File   = 0
	Wave   = 1
	FLAC   = 2
	MP3    = 3
	OGG    = 4
	MP4    = 5
	WebM   = 6
	Photo  = 7
	Bitmap = 8
	GIF    = 9
	PNG    = 10
	JPEG   = 11
	WebP   = 12
	PDF    = 13
	HTML   = 14
	Text   = 15
	Script = 16
	Config = 17
	LogFT  = 18
)

// File groups
const (
	FGOther = 0
	FGMusic = 1
	FGVideo = 2
	FGImage = 3
	FGBooks = 4
	FGTexts = 5
	FGDir   = 6
)

var typetogroup = map[int]int{
	Dir:    FGDir,
	File:   FGOther,
	Wave:   FGMusic,
	FLAC:   FGMusic,
	MP3:    FGMusic,
	OGG:    FGMusic,
	MP4:    FGVideo,
	WebM:   FGVideo,
	Photo:  FGImage,
	Bitmap: FGImage,
	GIF:    FGImage,
	PNG:    FGImage,
	JPEG:   FGImage,
	WebP:   FGImage,
	PDF:    FGBooks,
	HTML:   FGBooks,
	Text:   FGTexts,
	Script: FGTexts,
	Config: FGTexts,
	LogFT:  FGTexts,
}

var extset = map[string]int{
	// Audio
	".wav":  Wave,
	".flac": FLAC,
	".mp3":  MP3,
	".ogg":  OGG,

	// Video
	".mp4":  MP4,
	".webm": WebM,

	// Images
	".bmp":  Bitmap,
	".dib":  Bitmap,
	".gif":  GIF,
	".png":  PNG,
	".jpg":  JPEG,
	".jpe":  JPEG,
	".jpeg": JPEG,
	".webp": WebP,

	// Text
	".pdf":   PDF,
	".html":  HTML,
	".htm":   HTML,
	".shtml": HTML,
	".shtm":  HTML,
	".xhtml": HTML,
	".phtml": HTML,
	".hta":   HTML,
	".txt":   Text,
	".css":   Script,
	".js":    Script,
	".jsm":   Script,
	".vb":    Script,
	".vbs":   Script,
	".bat":   Script,
	".cmd":   Script,
	".sh":    Script,
	".mak":   Script,
	".iss":   Script,
	".nsi":   Script,
	".nsh":   Script,
	".bsh":   Script,
	".sql":   Script,
	".as":    Script,
	".mx":    Script,
	".php":   Script,
	".phpt":  Script,
	".java":  Script,
	".jsp":   Script,
	".asp":   Script,
	".lua":   Script,
	".tcl":   Script,
	".asm":   Script,
	".c":     Script,
	".h":     Script,
	".hpp":   Script,
	".hxx":   Script,
	".cpp":   Script,
	".cxx":   Script,
	".cc":    Script,
	".cs":    Script,
	".go":    Script,
	".r":     Script,
	".d":     Script,
	".pas":   Script,
	".inc":   Script,
	".py":    Script,
	".pyw":   Script,
	".pl":    Script,
	".pm":    Script,
	".plx":   Script,
	".rb":    Script,
	".rbw":   Script,
	".rc":    Script,
	".ps":    Script,
	".ini":   Config,
	".inf":   Config,
	".reg":   Config,
	".url":   Config,
	".xml":   Config,
	".xsml":  Config,
	".xsl":   Config,
	".xsd":   Config,
	".kml":   Config,
	".wsdl":  Config,
	".xlf":   Config,
	".xliff": Config,
	".yml":   Config,
	".cmake": Config,
	".vhd":   Config,
	".vhdl":  Config,
	".json":  Config,
	".log":   LogFT,
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
}

func (fp *FileProp) Setup(fi os.FileInfo) {
	fp.Name = fi.Name()
	fp.Size = fi.Size()
	fp.Time = UnixJS(fi.ModTime())
	if fi.IsDir() {
		fp.Type = Dir
	} else {
		fp.Type = extset[strings.ToLower(filepath.Ext(fp.Name))]
		if (fp.Type == JPEG && fp.Size > PhotoJPEG) || (fp.Type == WebP && fp.Size > PhotoWEBP) {
			fp.Type = Photo
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
	Paths []DirProp  `json:"paths"`
	Files []FileProp `json:"files"`
}

// Scan all available drives installed on local machine.
func getdrives() (drvs []DirProp) {
	for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		var name = string(drive) + ":"
		var f, err = os.Open(name)
		if err == nil {
			f.Close()
			var dp = DirProp{
				FileProp: FileProp{
					Name: name,
					Path: name + "/",
					Type: Dir,
				},
			}
			shrmux.RLock()
			if shr, ok := sharespath[dp.Path]; ok {
				dp.Pref = shr.Pref
			}
			shrmux.RUnlock()
			drvs = append(drvs, dp)
		}
	}

	root.Scan = UnixJS(time.Now())
	root.FGrp[FGDir] = len(drvs)
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

	var f *os.File
	f, err = os.Open(dirname)
	if err != nil {
		return
	}
	var fis []os.FileInfo
	fis, err = f.Readdir(-1)
	f.Close()
	if err != nil {
		return
	}
	var fgrp = [7]int{}
	var scan = UnixJS(time.Now())

scanprop:
	for _, fi := range fis {
		var fname = fi.Name()
		var fpath = dirname + fname
		var size = fi.Size()
		var ft int
		if fi.IsDir() {
			ft = Dir
			fpath += "/"
		} else {
			ft = extset[strings.ToLower(filepath.Ext(fname))]
			if (ft == JPEG && size > PhotoJPEG) || (ft == WebP && size > PhotoWEBP) {
				ft = Photo
			}
		}
		fgrp[typetogroup[ft]]++

		for _, pat := range hidden {
			if matched, _ := path.Match(pat, strings.ToLower(fname)); matched {
				continue scanprop
			}
		}

		var pref string
		shrmux.RLock()
		if shr, ok := sharespath[fpath]; ok {
			pref = shr.Pref
		}
		shrmux.RUnlock()
		var fp = FileProp{
			Name: fname,
			Path: fpath,
			Size: size,
			Time: UnixJS(fi.ModTime()),
			Type: ft,
			Pref: pref,
		}

		if ft == Dir {
			dcmux.RLock()
			var dc, ok = dircache[dirname+fname+"/"]
			dcmux.RUnlock()
			var dp = DirProp{
				FileProp: fp,
			}
			if ok {
				dp.Scan = dc.Scan
				dp.FGrp = dc.FGrp
			}
			ret.Paths = append(ret.Paths, dp)
		} else {
			ret.Files = append(ret.Files, fp)
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
		var dp = &DirProp{
			FileProp: FileProp{
				Name: fname,
				Path: dirname,
				Type: Dir,
			},
			Scan: scan,
			FGrp: fgrp,
		}

		dcmux.Lock()
		dircache[dirname] = dp
		dcmux.Unlock()
	}

	return
}

// The End.
