package joint

import (
	"io"
	"io/fs"
	"strings"

	"github.com/studio-b12/gowebdav"
)

type DavFileInfo = gowebdav.File

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

// DavJoint keeps gowebdav.Client object.
type DavJoint struct {
	key    string // URL to service, address + service route, i.e. https://user:pass@example.com/webdav/
	client *gowebdav.Client

	path string // truncated file path from full URL
	io.ReadCloser
	pos int64
	end int64
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

func (jnt *DavJoint) Busy() bool {
	return jnt.path != ""
}

func (jnt *DavJoint) Open(fpath string) (file fs.File, err error) {
	jnt.path = fpath
	return jnt, nil
}

func (jnt *DavJoint) Close() (err error) {
	jnt.path = ""
	if jnt.ReadCloser != nil {
		jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = 0
	jnt.end = 0
	PutJoint(jnt)
	return
}

func (jnt *DavJoint) Info(fpath string) (fi fs.FileInfo, err error) {
	return jnt.client.Stat(fpath)
}

func (jnt *DavJoint) ReadDir(fpath string) ([]fs.FileInfo, error) {
	return jnt.client.ReadDir(fpath)
}

func (jnt *DavJoint) Read(b []byte) (n int, err error) {
	if jnt.ReadCloser == nil {
		if jnt.ReadCloser, err = jnt.client.ReadStreamRange(jnt.path, jnt.pos, 0); err != nil {
			return
		}
	}
	n, err = jnt.ReadCloser.Read(b)
	jnt.pos += int64(n)
	return
}

func (jnt *DavJoint) Seek(offset int64, whence int) (abs int64, err error) {
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = jnt.pos + offset
	case io.SeekEnd:
		if jnt.end == 0 {
			var fi fs.FileInfo
			if fi, err = jnt.client.Stat(jnt.path); err != nil {
				return
			}
			jnt.end = fi.Size()
		}
		abs = jnt.end + offset
	default:
		err = ErrFtpWhence
		return
	}
	if abs < 0 {
		err = ErrFtpNegPos
		return
	}
	if abs != jnt.pos && jnt.ReadCloser != nil {
		jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = abs
	return
}

func (jnt *DavJoint) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		err = ErrFtpNegPos
		return
	}
	if off != jnt.pos && jnt.ReadCloser != nil {
		jnt.ReadCloser.Close()
		jnt.ReadCloser = nil
	}
	jnt.pos = off
	return jnt.Read(b)
}

func (jnt *DavJoint) Stat() (fi fs.FileInfo, err error) {
	fi, err = jnt.client.Stat(jnt.path)
	return
}

// The End.
