package joint

import (
	"io"
	"io/fs"

	iso "github.com/kdomanski/iso9660"
	"golang.org/x/text/encoding/charmap"
)

// IsoFile implements for ISO9660-file io.Reader, io.Seeker, io.Closer.
type IsoFile struct {
	extpath string // external path, to ISO9660-file disk image at local filesystem
	jnt     *IsoJoint
	*iso.File
	*io.SectionReader
}

func (f *IsoFile) Open(extpath, intpath string) (err error) {
	f.extpath = extpath
	if f.jnt, err = GetIsoJoint(extpath); err != nil {
		return
	}
	if f.File, err = f.jnt.OpenFile(intpath); err != nil {
		return
	}
	if sr := f.File.Reader(); sr != nil {
		f.SectionReader = sr.(*io.SectionReader)
	}
	return
}

func (f *IsoFile) Close() error {
	if f.jnt != nil {
		PutIsoJoint(f.extpath, f.jnt)
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

type DavFile struct {
	addr string // URL to service, address + service route, i.e. https://user:pass@example.com/webdav/
	path string // truncated file path from full URL
	jnt  *DavJoint
	io.ReadCloser
	pos int64
	end int64
}

func (f *DavFile) Open(davurl string) (err error) {
	var ok bool
	if f.addr, f.path, ok = GetDavPath(davurl); !ok {
		err = ErrNotFound
		return
	}
	if f.jnt, err = GetDavJoint(f.addr); err != nil {
		return
	}

	f.ReadCloser = nil
	f.pos, f.end = 0, 0
	return
}

func (f *DavFile) Close() (err error) {
	if f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	PutDavJoint(f.addr, f.jnt)
	return
}

func (f *DavFile) Read(b []byte) (n int, err error) {
	if f.ReadCloser == nil {
		if f.ReadCloser, err = f.jnt.client.ReadStreamRange(f.path, f.pos, 0); err != nil {
			return
		}
	}
	n, err = f.ReadCloser.Read(b)
	f.pos += int64(n)
	return
}

func (f *DavFile) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.pos + offset
	case io.SeekEnd:
		if f.end == 0 {
			var fi fs.FileInfo
			if fi, err = f.jnt.client.Stat(f.path); err != nil {
				return
			}
			f.end = fi.Size()
		}
		abs = f.end + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
		return
	}
	if abs != f.pos && f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	f.pos = abs
	return
}

func (f *DavFile) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		err = ErrFtpNegPos
		return
	}
	if off != f.pos && f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	f.pos = off
	return f.Read(b)
}

func (f *DavFile) Stat() (fi fs.FileInfo, err error) {
	fi, err = f.jnt.Stat(f.path)
	return
}

// The End.
