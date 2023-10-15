package hms

import (
	"io/fs"
	"path"
	"time"
)

// DiskPath contains disk full path and icon label.
type DiskPath struct {
	Path string `xorm:"'path'" json:"path" yaml:"path" xml:"path"`
	Name string `xorm:"'name'" json:"name" yaml:"name" xml:"name"`
}

// MakeFilePath creates DiskPath for file system path.
func MakeFilePath(syspath string) DiskPath {
	return DiskPath{
		Path: syspath,
		Name: path.Base(syspath),
	}
}

// FileProp is common file properties chunk.
type FileProp struct {
	Name string    `xorm:"'name'" json:"name" yaml:"name" xml:"name"`
	Type FT_t      `xorm:"'type'" json:"type" yaml:"type" xml:"type"` // do not omit empty
	Size int64     `xorm:"'size' default 0" json:"size,omitempty" yaml:"size,omitempty" xml:"size,omitempty"`
	Time time.Time `xorm:"'time' DateTime default 0" json:"time,omitempty" yaml:"time,omitempty" xml:"time,omitempty"`
}

// Setup fills fields from fs.FileInfo structure. Do not looks for share.
func (fp *FileProp) Setup(fi fs.FileInfo) {
	fp.Name = fi.Name()
	fp.Type = FTfile
	fp.Size = fi.Size()
	fp.Time = fi.ModTime()
}

// PuidProp encapsulated path unique ID value for some properties kit.
type PuidProp struct {
	PUID   Puid_t `xorm:"'puid'" json:"puid" yaml:"puid" xml:"puid,attr"`
	Free   bool   `xorm:"'free'" json:"free" yaml:"free" xml:"free,attr"`
	Shared bool   `xorm:"'shared'" json:"shared" yaml:"shared" xml:"shared,attr"`
	Static bool   `xorm:"'static'" json:"static" yaml:"static" xml:"static,attr"`
}

func (pp *PuidProp) Setup(session *Session, syspath string) {
	pp.PUID = PathStoreCache(session, syspath)
}

type ExtTag uint

const (
	TagThumb ExtTag = 1 << iota
	TagExif
	TagID3
)

type ExtProp struct {
	Tags    ExtTag        `json:"tags" yaml:"tags" xml:"tags"`
	Width   int           `json:"width,omitempty" yaml:"width,omitempty" xml:"width,omitempty"`
	Height  int           `json:"height,omitempty" yaml:"height,omitempty" xml:"height,omitempty"`
	Length  time.Duration `json:"length,omitempty" yaml:"length,omitempty" xml:"length,omitempty"`
	BitRate int           `xorm:"bitrate" json:"bitrate,omitempty" yaml:"bitrate,omitempty" xml:"bitrate,omitempty"`
}

// FileKit is common files properties kit.
type FileKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	TileProp `xorm:"extends" yaml:",inline"`
	ExtProp  `xorm:"extends" yaml:",inline"`
}

type FileGroup struct {
	FGother uint `xorm:"'other' default 0" json:"other,omitempty" yaml:"other,omitempty" xml,omitempty,attr:"other"`
	FGvideo uint `xorm:"'video' default 0" json:"video,omitempty" yaml:"video,omitempty" xml,omitempty,attr:"video"`
	FGaudio uint `xorm:"'audio' default 0" json:"audio,omitempty" yaml:"audio,omitempty" xml,omitempty,attr:"audio"`
	FGimage uint `xorm:"'image' default 0" json:"image,omitempty" yaml:"image,omitempty" xml,omitempty,attr:"image"`
	FGbooks uint `xorm:"'books' default 0" json:"books,omitempty" yaml:"books,omitempty" xml,omitempty,attr:"books"`
	FGtexts uint `xorm:"'texts' default 0" json:"texts,omitempty" yaml:"texts,omitempty" xml,omitempty,attr:"texts"`
	FGpacks uint `xorm:"'packs' default 0" json:"packs,omitempty" yaml:"packs,omitempty" xml,omitempty,attr:"packs"`
	FGgroup uint `xorm:"'group' default 0" json:"group,omitempty" yaml:"group,omitempty" xml,omitempty,attr:"group"`
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
	case FGgroup:
		return &fg.FGgroup
	default:
		return nil
	}
}

// Sum returns sum of all fields.
func (fg *FileGroup) Sum() uint {
	return fg.FGother + fg.FGvideo + fg.FGaudio + fg.FGimage + fg.FGbooks + fg.FGtexts + fg.FGpacks + fg.FGgroup
}

// IsZero used to check whether an object is zero to determine whether
// it should be omitted when marshaling to yaml.
func (fg *FileGroup) IsZero() bool {
	return fg.Sum() == 0
}

// DirProp is directory properties chunk.
type DirProp struct {
	Scan    time.Time `json:"scan,omitempty" yaml:"scan,omitempty" xml:"scan,omitempty"`                           // directory scanning time in UNIX format, milliseconds.
	FGrp    FileGroup `xorm:"extends" json:"fgrp,omitempty" yaml:"fgrp,flow,omitempty" xml:"fgrp,omitempty"`       // directory file groups counters.
	Latency int       `xorm:"default 0" json:"latency,omitempty" yaml:"latency,omitempty" xml:"latency,omitempty"` // drive connection latency in ms, or -1 on error
}

// DirKit is directory properties kit.
type DirKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	DirProp  `xorm:"extends" yaml:",inline"`
}

// The End.
