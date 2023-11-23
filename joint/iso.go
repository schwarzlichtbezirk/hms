package joint

import (
	"io"
	"io/fs"
	"strings"

	iso "github.com/kdomanski/iso9660"
	"golang.org/x/text/encoding/charmap"
)

// IsoJoint opens file with ISO9660 disk and prepares disk-structure
// to access to nested files.
type IsoJoint struct {
	key   string // external path, to ISO9660-file disk image at local filesystem
	file  RFile
	img   *iso.Image
	cache map[string]*iso.File
}

func (jnt *IsoJoint) Make(isopath string) (err error) {
	jnt.key = isopath
	if jnt.file, err = OpenFile(isopath); err != nil {
		return
	}
	if jnt.img, err = iso.OpenImage(jnt.file); err != nil {
		return
	}
	jnt.cache = map[string]*iso.File{}
	if jnt.cache[""], err = jnt.img.RootDir(); err != nil {
		return
	}
	return
}

func (jnt *IsoJoint) Cleanup() error {
	return jnt.file.Close()
}

func (jnt *IsoJoint) Key() string {
	return jnt.key
}

func (jnt *IsoJoint) Open(fpath string) (file RFile, err error) {
	var f = IsoFile{
		jnt: jnt,
	}
	if f.File, err = f.jnt.OpenFile(fpath); err != nil {
		return
	}
	if sr := f.File.Reader(); sr != nil {
		f.SectionReader = sr.(*io.SectionReader)
	}
	file = &f
	return
}

func (jnt *IsoJoint) OpenFile(intpath string) (file *iso.File, err error) {
	if file, ok := jnt.cache[intpath]; ok {
		return file, nil
	}

	var dec = charmap.Windows1251.NewDecoder()
	var curdir string
	var chunks = strings.Split(intpath, "/")
	file = jnt.cache[curdir] // get root directory
	for _, chunk := range chunks {
		if !file.IsDir() {
			err = ErrNotFound
			return
		}
		var curpath = joinfast(curdir, chunk)
		if f, ok := jnt.cache[curpath]; ok {
			file = f
		} else {
			var list []*iso.File
			if list, err = file.GetChildren(); err != nil {
				return
			}
			var found = false
			for _, file = range list {
				var name, _ = dec.String(file.Name())
				jnt.cache[joinfast(curdir, name)] = file
				if name == chunk {
					found = true
					break
				}
			}
			if !found {
				err = ErrNotFound
				return
			}
		}
		curdir = curpath
	}
	return
}

func (jnt *IsoJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	var file *iso.File
	if file, err = jnt.OpenFile(fpath); err != nil {
		return
	}
	fi = &IsoFile{
		File: file,
	}
	return
}

func (jnt *IsoJoint) ReadDir(fpath string) (ret []fs.FileInfo, err error) {
	var f RFile
	if f, err = jnt.Open(fpath); err != nil {
		return
	}
	defer f.Close()
	var files []*iso.File
	if files, err = f.(*IsoFile).GetChildren(); err != nil {
		return
	}
	ret = make([]fs.FileInfo, len(files))
	for i, file := range files {
		ret[i] = &IsoFile{
			File: file,
		}
	}
	return
}

// IsoFile implements for ISO9660-file io.Reader, io.Seeker, io.Closer.
type IsoFile struct {
	jnt *IsoJoint
	*iso.File
	*io.SectionReader
}

func (f *IsoFile) Close() error {
	if f.jnt != nil {
		PutJoint(f.jnt)
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
