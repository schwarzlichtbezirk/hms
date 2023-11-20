package joint

import (
	"io"
	"io/fs"

	iso "github.com/kdomanski/iso9660"
	"golang.org/x/text/encoding/charmap"
)

// IsoFile implements for ISO9660-file io.Reader, io.Seeker, io.Closer.
type IsoFile struct {
	jnt *IsoJoint
	*iso.File
	*io.SectionReader
}

func (f *IsoFile) Close() error {
	if f.jnt != nil {
		PutIsoJoint(f.jnt)
	}
	return nil
}

func (f *IsoFile) Name() string {
	var dec = charmap.Windows1251.NewDecoder()
	var name, _ = dec.String(f.File.Name())
	return name
}

func (f *IsoFile) Size() int64 {
	return f.File.Size()
}

func (f *IsoFile) Stat() (fs.FileInfo, error) {
	return f.File, nil
}

func (f *IsoFile) Sys() interface{} {
	return f
}

// The End.
