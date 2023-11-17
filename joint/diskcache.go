package joint

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"

	"github.com/jlaffaye/ftp"
	iso "github.com/kdomanski/iso9660"
	"github.com/pkg/sftp"
	"github.com/studio-b12/gowebdav"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/charmap"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrUnexpDir = errors.New("unexpected directory instead the file")
	ErrNotIso   = errors.New("filesystem is not ISO9660")
)

// IsStatic returns whether file info refers to content
// that can not be modified or moved.
func IsStatic(fi fs.FileInfo) (static bool) {
	if static = fi == nil; static {
		return
	}
	if _, static = fi.(*IsoFile); static {
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

// DiskCache implements cache with opened joints to some disk resource.
type DiskCache[T MakeCloser] struct {
	cache  []T
	expire []*time.Timer
	mux    sync.Mutex
}

func (dc *DiskCache[T]) Count() int {
	dc.mux.Lock()
	defer dc.mux.Unlock()
	return len(dc.cache)
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
	dc.expire = append(dc.expire, time.AfterFunc(cfg.Cfg.DiskCacheExpire, func() {
		if val, ok := dc.Peek(); ok {
			val.Close()
		}
	}))
}

// IsoJoint opens file with ISO9660 disk and prepares disk-structure
// to access to nested files.
type IsoJoint struct {
	file  RFile
	img   *iso.Image
	cache map[string]*iso.File
}

func (d *IsoJoint) Make(isopath string) (err error) {
	if d.file, err = OpenFile(isopath); err != nil {
		return
	}
	if d.img, err = iso.OpenImage(d.file); err != nil {
		return
	}
	d.cache = map[string]*iso.File{}
	if d.cache[""], err = d.img.RootDir(); err != nil {
		return
	}
	return
}

func (d *IsoJoint) Close() error {
	return d.file.Close()
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

func (d *IsoJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	var file *iso.File
	if file, err = d.OpenFile(fpath); err != nil {
		return
	}
	fi = &IsoFile{
		File: file,
	}
	return
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
	if chunks[len(chunks)-1] == "" {
		chunks = chunks[:len(chunks)-1]
	}
	for _, chunk := range chunks {
		dpath += chunk + "/"
		var client = gowebdav.NewClient(dpath, "", "")
		if fi, err := client.Stat(""); err == nil && fi.IsDir() {
			PutDavJoint(dpath, &DavJoint{
				client: client,
			})
			DavPath[addr] = dpath
			ok = true
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
	if d.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.Cfg.DialTimeout)); err != nil {
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
