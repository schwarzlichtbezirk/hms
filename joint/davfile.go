package joint

import (
	"io"
	"io/fs"
)

type DavFile struct {
	jnt  *DavJoint
	path string // truncated file path from full URL
	io.ReadCloser
	pos int64
	end int64
}

func (f *DavFile) Close() (err error) {
	if f.ReadCloser != nil {
		f.ReadCloser.Close()
		f.ReadCloser = nil
	}
	PutDavJoint(f.jnt)
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
