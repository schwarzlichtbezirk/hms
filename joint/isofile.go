package joint

import (
	"io"
	"io/fs"
	"os"
	"path"

	"golang.org/x/text/encoding/charmap"
)

// IsoFile implements for ISO9660-file io.Reader, io.Seeker, io.Closer.
type IsoFile struct {
	isofile string // path to ISO9660-file disk image at local filesystem
	fpath   string // path inside of disk image
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

type DavFile struct {
	addr string // URL to service, address + service route, i.e. https://user:pass@example.com/webdav/
	path string // truncated file path from full URL
	d    *DavJoint
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
	if f.d, err = GetDavJoint(f.addr); err != nil {
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
	PutDavJoint(f.addr, f.d)
	return
}

func (f *DavFile) Read(b []byte) (n int, err error) {
	if f.ReadCloser == nil {
		if f.ReadCloser, err = f.d.client.ReadStreamRange(f.path, f.pos, 0); err != nil {
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
			if fi, err = f.d.client.Stat(f.path); err != nil {
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
	}
	if abs != f.pos && f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	f.pos = abs
	return
}

func (f *DavFile) Stat() (fi fs.FileInfo, err error) {
	fi, err = f.d.Stat(f.path)
	return
}

// The End.
