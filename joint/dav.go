package joint

import (
	"io"
	"io/fs"
	"strings"

	"github.com/studio-b12/gowebdav"
)

type DavFileStat = gowebdav.File

// DavJoint keeps gowebdav.Client object.
type DavJoint struct {
	key    string // URL to service, address + service route, i.e. https://user:pass@example.com/webdav/
	client *gowebdav.Client
}

func (jnt *DavJoint) Make(urladdr string) (err error) {
	jnt.key = urladdr
	jnt.client = gowebdav.NewClient(urladdr, "", "") // user & password gets from URL
	err = jnt.client.Connect()
	return
}

func (jnt *DavJoint) Cleanup() error {
	return nil
}

func (jnt *DavJoint) Key() string {
	return jnt.key
}

func (jnt *DavJoint) Open(fpath string) (file RFile, err error) {
	return &DavFile{
		jnt:  jnt,
		path: fpath,
	}, nil
}

func (jnt *DavJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	return jnt.client.Stat(fpath)
}

func (jnt *DavJoint) ReadDir(fpath string) ([]fs.FileInfo, error) {
	return jnt.client.ReadDir(fpath)
}

// DavPath is map of WebDAV servises root paths by services URLs.
var DavPath = map[string]string{}

func GetDavPath(davurl string) (dpath, fpath string, ok bool) {
	defer func() {
		if ok && dpath != davurl+"/" {
			fpath = davurl[len(dpath):]
		}
	}()
	var addr, route = SplitUrl(davurl)
	if dpath, ok = DavPath[addr]; ok {
		return
	}

	dpath = addr
	var chunks = strings.Split("/"+route, "/")
	if chunks[len(chunks)-1] == "" {
		chunks = chunks[:len(chunks)-1]
	}
	for _, chunk := range chunks {
		dpath += chunk + "/"
		var client = gowebdav.NewClient(dpath, "", "")
		if fi, err := client.Stat(""); err == nil && fi.IsDir() {
			PutJoint(&DavJoint{
				key:    dpath,
				client: client,
			})
			DavPath[addr] = dpath
			ok = true
			return
		}
	}
	return
}

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
	PutJoint(f.jnt)
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
