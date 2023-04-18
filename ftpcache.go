package hms

import (
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
)

type MakeCloser interface {
	Make(urladdr string) error
	io.Closer
}

type CloudCache[T MakeCloser] struct {
	cache  []T
	expire []*time.Timer
	mux    sync.Mutex
}

func (cc *CloudCache[T]) Close() (err error) {
	cc.mux.Lock()
	defer cc.mux.Unlock()

	for _, t := range cc.expire {
		t.Stop()
	}
	cc.expire = nil

	for _, t := range cc.cache {
		if err1 := t.Close(); err1 != nil {
			err = err1
		}
	}
	cc.cache = nil
	return
}

func (cc *CloudCache[T]) Peek() (val T, ok bool) {
	cc.mux.Lock()
	defer cc.mux.Unlock()
	var l = len(cc.cache)
	if l > 0 {
		cc.expire[0].Stop()
		cc.expire = cc.expire[1:]
		val = cc.cache[0]
		cc.cache = cc.cache[1:]
		ok = true
	}
	return
}

func (cc *CloudCache[T]) Put(val T) {
	cc.mux.Lock()
	defer cc.mux.Unlock()
	cc.cache = append(cc.cache, val)
	cc.expire = append(cc.expire, time.AfterFunc(cfg.DiskCacheExpire, func() {
		if val, ok := cc.Peek(); ok {
			val.Close()
		}
	}))
}

type FtpConn struct {
	conn *ftp.ServerConn
}

func (c *FtpConn) Make(urladdr string) (err error) {
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	if c.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	if err = c.conn.Login(u.User.Username(), pass); err != nil {
		return
	}
	return
}

func (c *FtpConn) Close() error {
	return c.conn.Quit()
}

var FtpCaches = map[string]*CloudCache[*FtpConn]{}

func GetFtpConn(ftpaddr string) (conn *ftp.ServerConn, err error) {
	var ok bool
	var cc *CloudCache[*FtpConn]
	if cc, ok = FtpCaches[ftpaddr]; !ok {
		cc = &CloudCache[*FtpConn]{}
		FtpCaches[ftpaddr] = cc
	}
	var fc *FtpConn
	if fc, ok = cc.Peek(); !ok {
		fc = &FtpConn{}
		err = fc.Make(ftpaddr)
	}
	conn = fc.conn
	return
}

func PutFtpConn(ftpaddr string, conn *ftp.ServerConn) {
	var ok bool
	var cc *CloudCache[*FtpConn]
	if cc, ok = FtpCaches[ftpaddr]; !ok {
		cc = &CloudCache[*FtpConn]{}
		FtpCaches[ftpaddr] = cc
	}
	cc.Put(&FtpConn{
		conn: conn,
	})
}

// The End.
