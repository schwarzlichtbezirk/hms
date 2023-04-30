package hms

import (
	"io"
	"io/fs"
	"os"
	"path"

	"golang.org/x/text/encoding/charmap"
)

// IsoFile implements for ISO9660-file io.Reader, io.Seeker, io.Closer.
type IsoFile struct {
	isofile string
	fpath   string
	d       *IsoJoint
	io.ReadSeekCloser
}

func (f *IsoFile) Open(isofile, fpath string) (err error) {
	f.isofile, f.fpath = isofile, fpath
	if f.d, err = GetIsoJoint(isofile); err != nil {
		return
	}
	var enc = charmap.Windows1251.NewEncoder()
	fpath, _ = enc.String(fpath)
	if f.ReadSeekCloser, err = f.d.fs.OpenFile(fpath, os.O_RDONLY); err != nil {
		return
	}
	return
}

func (f *IsoFile) Close() (err error) {
	err = f.ReadSeekCloser.Close()
	PutIsoJoint(f.isofile, f.d)
	return
}

func (f *IsoFile) Stat() (fi fs.FileInfo, err error) {
	var enc = charmap.Windows1251.NewEncoder()
	var fpath, _ = enc.String(f.fpath)

	var list []fs.FileInfo
	if list, err = f.d.fs.ReadDir(path.Dir(fpath)); err != nil {
		return
	}

	var fname = path.Base(fpath)
	for _, fi = range list {
		if fi.Name() == fname {
			return IsoFileInfo{fi}, nil
		}
	}
	return nil, ErrNotFound
}

// IsoFileInfo is wrapper to convert file names in code page 1251 to UTF.
type IsoFileInfo struct {
	fs.FileInfo
}

func (fi IsoFileInfo) Name() string {
	var dec = charmap.Windows1251.NewDecoder()
	var name, _ = dec.String(fi.FileInfo.Name())
	return name
}

// The End.
