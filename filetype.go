package hms

import (
	"path"
	"strings"
)

// FT_t is enum type for properties file types.
type FT_t int

// File types.
const (
	FTunk  FT_t = 0 // unknown file type
	FTfile FT_t = 1
	FTdir  FT_t = 2
	FTdrv  FT_t = 3
	FTctgr FT_t = 4
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
	return extgrp[ext] == FGimage
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

// The End.
