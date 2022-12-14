package hms

import (
	"io/fs"
	"path"
)

func GetPropPUID(p any) (Puid_t, bool) {
	switch v := p.(type) {
	case *FileKit:
		return v.PUID, true
	case *DirKit:
		return v.PUID, true
	case *ExifKit:
		return v.PUID, true
	case *TagKit:
		return v.PUID, true
	default:
		return 0, false
	}
}

func GetPropSize(p any) (int64, bool) {
	switch v := p.(type) {
	case *FileKit:
		return v.Size, true
	case *DirKit:
		return v.Size, true
	case *ExifKit:
		return v.Size, true
	case *TagKit:
		return v.Size, true
	default:
		return 0, false
	}
}

func GetPropType(p any) (FT_t, bool) {
	switch v := p.(type) {
	case *FileKit:
		return v.Type, true
	case *DirKit:
		return v.Type, true
	case *ExifKit:
		return v.Type, true
	case *TagKit:
		return v.Type, true
	default:
		return 0, false
	}
}

// FileProp is common file properties chunk.
type FileProp struct {
	Name string `xorm:"'name'" json:"name" yaml:"name" xml:"name"`
	Type FT_t   `xorm:"'type'" json:"type" yaml:"type" xml:"type"` // do not omit empty
	Size int64  `xorm:"'size' default 0" json:"size,omitempty" yaml:"size,omitempty" xml:"size,omitempty"`
	Time Unix_t `xorm:"'time' default 0" json:"time,omitempty" yaml:"time,omitempty" xml:"time,omitempty"`
}

// Setup fills fields from fs.FileInfo structure. Do not looks for share.
func (fp *FileProp) Setup(fi fs.FileInfo) {
	fp.Name = path.Clean(fi.Name())
	fp.Type = FTfile
	fp.Size = fi.Size()
	fp.Time = UnixJS(fi.ModTime())
}

// FileKit is common files properties kit.
type FileKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	TmbProp  `xorm:"extends" yaml:",inline"`
}

// Setup calls nested structures setups.
func (fk *FileKit) Setup(session *Session, syspath string, fi fs.FileInfo) {
	fk.FileProp.Setup(fi)
	fk.PuidProp.Setup(session, syspath)
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

// DirProp is directory properties chunk.
type DirProp struct {
	Scan    Unix_t    `xorm:"default 0" json:"scan,omitempty" yaml:"scan,omitempty" xml:"scan,omitempty"`          // directory scanning time in UNIX format, milliseconds.
	FGrp    FileGroup `xorm:"extends" json:"fgrp,omitempty" yaml:"fgrp,flow,omitempty" xml:"fgrp,omitempty"`       // directory file groups counters.
	Latency int       `xorm:"default 0" json:"latency,omitempty" yaml:"latency,omitempty" xml:"latency,omitempty"` // drive connection latency in ms, or -1 on error
}

// DirKit is directory properties kit.
type DirKit struct {
	PuidProp `xorm:"extends" yaml:",inline"`
	FileProp `xorm:"extends" yaml:",inline"`
	DirProp  `xorm:"extends" yaml:",inline"`
}

// Setup fills fields with given path. Do not looks for share.
func (dk *DirKit) Setup(session *Session, syspath string) {
	dk.Name = path.Base(syspath)
	dk.Type = FTdir
	dk.PuidProp.Setup(session, syspath)
	if dp, ok := DirStoreGet(session, dk.PUID); ok {
		dk.DirProp = dp
	}
}

// MakeProp is file properties factory.
func MakeProp(syspath string, fi fs.FileInfo) any {
	var session = xormEngine.NewSession()
	defer session.Close()

	if fi.IsDir() {
		var dk DirKit
		dk.Setup(session, syspath)
		return &dk
	}
	var ext = GetFileExt(syspath)
	if IsTypeID3(ext) {
		var tk TagKit
		tk.Setup(session, syspath, fi)
		return &tk
	} else if IsTypeEXIF(ext) {
		var ek ExifKit
		ek.Setup(session, syspath, fi)
		return &ek
	} else {
		var fk FileKit
		fk.Setup(session, syspath, fi)
		return &fk
	}
}

// The End.
