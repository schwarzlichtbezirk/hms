package hms

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/jlaffaye/ftp"
	"golang.org/x/text/encoding/charmap"
)

type MakeCloser interface {
	Make(urladdr string) error
	io.Closer
}

type File interface {
	io.ReadSeekCloser
	Stat() (fs.FileInfo, error)
}

type DiskCache[T MakeCloser] struct {
	cache  []T
	expire []*time.Timer
	mux    sync.Mutex
}

func (dc *DiskCache[T]) Close() (err error) {
	dc.mux.Lock()
	defer dc.mux.Unlock()

	for _, t := range dc.expire {
		t.Stop()
	}
	dc.expire = nil

	for _, t := range dc.cache {
		if err1 := t.Close(); err1 != nil {
			err = err1
		}
	}
	dc.cache = nil
	return
}

func (dc *DiskCache[T]) Peek() (val T, ok bool) {
	dc.mux.Lock()
	defer dc.mux.Unlock()
	var l = len(dc.cache)
	if l > 0 {
		dc.expire[0].Stop()
		dc.expire = dc.expire[1:]
		val = dc.cache[0]
		dc.cache = dc.cache[1:]
		ok = true
	}
	return
}

func (dc *DiskCache[T]) Put(val T) {
	dc.mux.Lock()
	defer dc.mux.Unlock()
	dc.cache = append(dc.cache, val)
	dc.expire = append(dc.expire, time.AfterFunc(cfg.DiskCacheExpire, func() {
		if val, ok := dc.Peek(); ok {
			val.Close()
		}
	}))
}

var (
	ErrNotIso = errors.New("filesystem is not ISO9660")
)

type IsoJoint struct {
	file *os.File
	fs   *iso9660.FileSystem
}

func (d *IsoJoint) Make(isopath string) (err error) {
	var disk *disk.Disk
	if disk, err = diskfs.Open(isopath, diskfs.WithOpenMode(diskfs.ReadOnly)); err != nil {
		return
	}
	d.file = disk.File
	var fs filesystem.FileSystem
	if fs, err = disk.GetFilesystem(0); err != nil { // assuming it is the whole disk, so partition = 0
		disk.File.Close()
		return
	}
	var ok bool
	if d.fs, ok = fs.(*iso9660.FileSystem); !ok {
		err = ErrNotIso
		return
	}
	return
}

func (d *IsoJoint) Close() error {
	return d.file.Close()
}

func (d *IsoJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	var enc = charmap.Windows1251.NewEncoder()
	fpath, _ = enc.String(fpath)

	var list []fs.FileInfo
	if list, err = d.fs.ReadDir(path.Dir(fpath)); err != nil {
		return
	}

	var fname = path.Base(fpath)
	for _, fi = range list {
		if fi.Name() == fname {
			return &IsoFileInfo{fi}, nil
		}
	}
	return nil, ErrNotFound
}

var IsoCaches = map[string]*DiskCache[*IsoJoint]{}

func GetIsoJoint(isopath string) (d *IsoJoint, err error) {
	var ok bool
	var dc *DiskCache[*IsoJoint]
	if dc, ok = IsoCaches[isopath]; !ok {
		dc = &DiskCache[*IsoJoint]{}
		IsoCaches[isopath] = dc
	}
	if d, ok = dc.Peek(); !ok {
		d = &IsoJoint{}
		err = d.Make(isopath)
	}
	return
}

func PutIsoJoint(isopath string, d *IsoJoint) {
	var ok bool
	var dc *DiskCache[*IsoJoint]
	if dc, ok = IsoCaches[isopath]; !ok {
		dc = &DiskCache[*IsoJoint]{}
		IsoCaches[isopath] = dc
	}
	dc.Put(d)
}

type FtpJoint struct {
	conn *ftp.ServerConn
	pwd  string
}

func (d *FtpJoint) Make(urladdr string) (err error) {
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	if d.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.DialTimeout)); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	if err = d.conn.Login(u.User.Username(), pass); err != nil {
		return
	}
	d.pwd = FtpPwd(u.Host, d.conn)
	return
}

func (d *FtpJoint) Close() error {
	return d.conn.Quit()
}

func (d *FtpJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	var ent *ftp.Entry
	if ent, err = d.conn.GetEntry(path.Join(d.pwd, fpath)); err != nil {
		return
	}
	fi = &FtpFileInfo{
		ent,
	}
	return
}

var FtpCaches = map[string]*DiskCache[*FtpJoint]{}

func GetFtpJoint(ftpaddr string) (d *FtpJoint, err error) {
	var ok bool
	var dc *DiskCache[*FtpJoint]
	if dc, ok = FtpCaches[ftpaddr]; !ok {
		dc = &DiskCache[*FtpJoint]{}
		FtpCaches[ftpaddr] = dc
	}
	if d, ok = dc.Peek(); !ok {
		d = &FtpJoint{}
		err = d.Make(ftpaddr)
	}
	return
}

func PutFtpJoint(ftpaddr string, d *FtpJoint) {
	var ok bool
	var dc *DiskCache[*FtpJoint]
	if dc, ok = FtpCaches[ftpaddr]; !ok {
		dc = &DiskCache[*FtpJoint]{}
		FtpCaches[ftpaddr] = dc
	}
	dc.Put(d)
}

// The End.
