package hms

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"github.com/studio-b12/gowebdav"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/charmap"
)

// IsStatic returns whether file info refers to content
// that can not be modified or moved.
func IsStatic(fi fs.FileInfo) (static bool) {
	if static = fi == nil; static {
		return
	}
	if _, static = fi.(IsoFileInfo); static {
		return
	}
	if _, static = fi.(gowebdav.File); static {
		return
	}
	if _, static = fi.(FtpFileInfo); static {
		return
	}
	if sys := fi.Sys(); sys != nil {
		if _, static = sys.(*sftp.FileStat); static {
			return
		}
	}
	return
}

// MakeCloser describes structure that can be initialized
// by some resource path, and can be closed.
type MakeCloser interface {
	Make(urladdr string) error
	io.Closer
}

// File combines fs.File interface and io.Seeker interface.
type File interface {
	io.ReadSeekCloser
	Stat() (fs.FileInfo, error)
}

// DiskCache implements cache with opened joints to some disk resource.
type DiskCache[T MakeCloser] struct {
	cache  []T
	expire []*time.Timer
	mux    sync.Mutex
}

// Close performs close-call to all cached disk joints.
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

// Peek retrieves cached disk joint, and returns ok if it has.
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

// Put disk joint to cache.
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

// IsoJoint opens file with ISO9660 disk and prepares disk-structure
// to access to nested files.
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
			return IsoFileInfo{fi}, nil
		}
	}
	return nil, ErrNotFound
}

// IsoCaches is map with ISO9660-disks joints.
// Each key is path to ISO-disk, value - cached for this disk list of joints.
var IsoCaches = map[string]*DiskCache[*IsoJoint]{}

// GetIsoJoint gets cached joint for given path to ISO-disk,
// or creates new one.
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

// PutIsoJoint puts to cache joint for ISO-disk with given path.
func PutIsoJoint(isopath string, d *IsoJoint) {
	var ok bool
	var dc *DiskCache[*IsoJoint]
	if dc, ok = IsoCaches[isopath]; !ok {
		dc = &DiskCache[*IsoJoint]{}
		IsoCaches[isopath] = dc
	}
	dc.Put(d)
}

// DavJoint keeps gowebdav.Client object.
type DavJoint struct {
	client *gowebdav.Client
}

func (d *DavJoint) Make(urladdr string) (err error) {
	d.client = gowebdav.NewClient(urladdr, "", "") // user & password gets from URL
	err = d.client.Connect()
	return
}

func (d *DavJoint) Close() error {
	return nil
}

func (d *DavJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	return d.client.Stat(fpath)
}

// DavCaches is map of gowebdav.Client joints.
// Each key is URL of WebDAV service, value - cached for this service list of joints.
var DavCaches = map[string]*DiskCache[*DavJoint]{}

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
	for _, chunk := range chunks {
		dpath += chunk + "/"
		var client = gowebdav.NewClient(dpath, "", "")
		if ok = client.Connect() == nil; ok {
			PutDavJoint(dpath, &DavJoint{
				client: client,
			})
			DavPath[addr] = dpath
			return
		}
	}
	return
}

// GetDavJoint gets cached joint for given URL to WebDAV service,
// or creates new one.
func GetDavJoint(davurl string) (d *DavJoint, err error) {
	var ok bool
	var dc *DiskCache[*DavJoint]
	if dc, ok = DavCaches[davurl]; !ok {
		dc = &DiskCache[*DavJoint]{}
		DavCaches[davurl] = dc
	}
	if d, ok = dc.Peek(); !ok {
		d = &DavJoint{}
		err = d.Make(davurl)
	}
	return
}

// PutDavJoint puts to cache joint for WebDAV service with given URL.
func PutDavJoint(davurl string, d *DavJoint) {
	var ok bool
	var dc *DiskCache[*DavJoint]
	if dc, ok = DavCaches[davurl]; !ok {
		dc = &DiskCache[*DavJoint]{}
		DavCaches[davurl] = dc
	}
	dc.Put(d)
}

// FtpJoint create connection to FTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
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
	fi = FtpFileInfo{
		ent,
	}
	return
}

// FtpCaches is map with FTP-joints.
// Each key is FTP-server address, value - cached on it's server list of joints.
var FtpCaches = map[string]*DiskCache[*FtpJoint]{}

// GetFtpJoint gets cached joint for given address to FTP-server,
// or creates new one.
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

// PutSftpJoint puts to cache joint for FTP-server with given address.
func PutFtpJoint(ftpaddr string, d *FtpJoint) {
	var ok bool
	var dc *DiskCache[*FtpJoint]
	if dc, ok = FtpCaches[ftpaddr]; !ok {
		dc = &DiskCache[*FtpJoint]{}
		FtpCaches[ftpaddr] = dc
	}
	dc.Put(d)
}

// SftpJoint create SSH-connection to SFTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
type SftpJoint struct {
	conn   *ssh.Client
	client *sftp.Client
	pwd    string
}

func (d *SftpJoint) Make(urladdr string) (err error) {
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	var config = &ssh.ClientConfig{
		User: u.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if d.conn, err = ssh.Dial("tcp", u.Host, config); err != nil {
		return
	}
	if d.client, err = sftp.NewClient(d.conn); err != nil {
		return
	}
	d.pwd = SftpPwd(u.Host, d.client)
	return
}

func (d *SftpJoint) Close() (err error) {
	err = d.client.Close()
	if err1 := d.conn.Close(); err1 != nil {
		err = err1
	}
	return
}

func (d *SftpJoint) Stat(fpath string) (fs.FileInfo, error) {
	return d.client.Stat(path.Join(d.pwd, fpath))
}

// SftpCaches is map with SFTP-joints.
// Each key is SFTP-server address, value - cached on it's server list of joints.
var SftpCaches = map[string]*DiskCache[*SftpJoint]{}

// GetSftpJoint gets cached joint for given address to SFTP-server,
// or creates new one.
func GetSftpJoint(sftpaddr string) (d *SftpJoint, err error) {
	var ok bool
	var dc *DiskCache[*SftpJoint]
	if dc, ok = SftpCaches[sftpaddr]; !ok {
		dc = &DiskCache[*SftpJoint]{}
		SftpCaches[sftpaddr] = dc
	}
	if d, ok = dc.Peek(); !ok {
		d = &SftpJoint{}
		err = d.Make(sftpaddr)
	}
	return
}

// PutSftpJoint puts to cache joint for SFTP-server with given address.
func PutSftpJoint(sftpaddr string, d *SftpJoint) {
	var ok bool
	var dc *DiskCache[*SftpJoint]
	if dc, ok = SftpCaches[sftpaddr]; !ok {
		dc = &DiskCache[*SftpJoint]{}
		SftpCaches[sftpaddr] = dc
	}
	dc.Put(d)
}

// The End.
