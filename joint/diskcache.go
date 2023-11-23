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

// Joint describes interface with joint to some file system provider.
type Joint interface {
	Make(urladdr string) error             // establish connection to file system provider
	Cleanup() error                        // close connection to file system provider
	Key() string                           // key path to resource
	Open(string) (RFile, error)            // open file with local file path
	Stat(string) (fs.FileInfo, error)      // returns file state pointed by local file path
	ReadDir(string) ([]fs.FileInfo, error) // read directory pointed by local file path
}

// JointCache implements cache with opened joints to some file system resource.
type JointCache struct {
	cache  []Joint
	expire []*time.Timer
	mux    sync.Mutex
}

func (jc *JointCache) Count() int {
	jc.mux.Lock()
	defer jc.mux.Unlock()
	return len(jc.cache)
}

// Close performs close-call to all cached disk joints.
func (jc *JointCache) Close() (err error) {
	jc.mux.Lock()
	defer jc.mux.Unlock()

	for _, t := range jc.expire {
		t.Stop()
	}
	jc.expire = nil

	for _, t := range jc.cache {
		if err1 := t.Cleanup(); err1 != nil {
			err = err1
		}
	}
	jc.cache = nil
	return
}

// Get retrieves cached disk joint, and returns ok if it has.
func (jc *JointCache) Get() (val Joint, ok bool) {
	jc.mux.Lock()
	defer jc.mux.Unlock()
	var l = len(jc.cache)
	if l > 0 {
		jc.expire[0].Stop()
		jc.expire = jc.expire[1:]
		val = jc.cache[0]
		jc.cache = jc.cache[1:]
		ok = true
	}
	return
}

// Put disk joint to cache.
func (jc *JointCache) Put(val Joint) {
	jc.mux.Lock()
	defer jc.mux.Unlock()
	jc.cache = append(jc.cache, val)
	jc.expire = append(jc.expire, time.AfterFunc(cfg.Cfg.DiskCacheExpire, func() {
		if val, ok := jc.Get(); ok {
			val.Cleanup()
		}
	}))
}

// JointPool is map with joint caches.
// Each key is path to file system resource,
// value - cached for this resource list of joints.
var JointPool = map[string]*JointCache{}

// GetIsoJoint gets cached joint for given key path,
// or creates new one on given template.
func GetJoint(isopath string, tmp Joint) (jnt Joint, err error) {
	var ok bool
	var jc *JointCache
	if jc, ok = JointPool[isopath]; !ok {
		jc = &JointCache{}
		JointPool[isopath] = jc
	}
	if jnt, ok = jc.Get(); !ok {
		jnt = tmp
		err = jnt.Make(isopath)
	}
	return
}

// PutJoint puts to cache given joint.
func PutJoint(jnt Joint) {
	var ok bool
	var jc *JointCache
	if jc, ok = JointPool[jnt.Key()]; !ok {
		jc = &JointCache{}
		JointPool[jnt.Key()] = jc
	}
	jc.Put(jnt)
}

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

// FtpJoint create connection to FTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
type FtpJoint struct {
	key  string // address of FTP-service, i.e. ftp://user:pass@example.com
	conn *ftp.ServerConn
	pwd  string
}

func (jnt *FtpJoint) Make(urladdr string) (err error) {
	jnt.key = urladdr
	var u *url.URL
	if u, err = url.Parse(urladdr); err != nil {
		return
	}
	if jnt.conn, err = ftp.Dial(u.Host, ftp.DialWithTimeout(cfg.Cfg.DialTimeout)); err != nil {
		return
	}
	var pass, _ = u.User.Password()
	if err = jnt.conn.Login(u.User.Username(), pass); err != nil {
		return
	}
	jnt.pwd = FtpPwd(u.Host, jnt.conn)
	return
}

func (jnt *FtpJoint) Cleanup() error {
	return jnt.conn.Quit()
}

func (jnt *FtpJoint) Key() string {
	return jnt.key
}

// Opens new connection for any some one file with given full FTP URL.
// FTP-connection can serve only one file by the time, so it can not
// be used for parallel reading group of files.
func (jnt *FtpJoint) Open(fpath string) (file RFile, err error) {
	return &FtpFile{
		jnt:  jnt,
		path: fpath,
	}, nil
}

func (jnt *FtpJoint) Stat(fpath string) (fi fs.FileInfo, err error) {
	var ent *ftp.Entry
	if ent, err = jnt.conn.GetEntry(path.Join(jnt.pwd, fpath)); err != nil {
		return
	}
	fi = FtpFileInfo{
		ent,
	}
	return
}

func (jnt *FtpJoint) ReadDir(fpath string) (ret []fs.FileInfo, err error) {
	fpath = FtpEscapeBrackets(path.Join(jnt.pwd, fpath))
	var entries []*ftp.Entry
	if entries, err = jnt.conn.List(fpath); err != nil {
		return
	}
	ret = make([]fs.FileInfo, 0, len(entries))
	for _, ent := range entries {
		if ent.Name != "." && ent.Name != ".." {
			ret = append(ret, FtpFileInfo{ent})
		}
	}
	return
}

// SftpJoint create SSH-connection to SFTP-server, login with provided by
// given URL credentials, and gets a once current directory.
// Joint can be used for followed files access.
type SftpJoint struct {
	key    string // address of SFTP-service, i.e. sftp://user:pass@example.com
	conn   *ssh.Client
	client *sftp.Client
	pwd    string
}

func (jnt *SftpJoint) Make(urladdr string) (err error) {
	jnt.key = urladdr
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
	if jnt.conn, err = ssh.Dial("tcp", u.Host, config); err != nil {
		return
	}
	if jnt.client, err = sftp.NewClient(jnt.conn); err != nil {
		return
	}
	jnt.pwd = SftpPwd(u.Host, jnt.client)
	return
}

func (jnt *SftpJoint) Cleanup() (err error) {
	err = jnt.client.Close()
	if err1 := jnt.conn.Close(); err1 != nil {
		err = err1
	}
	return
}

func (jnt *SftpJoint) Key() string {
	return jnt.key
}

// Opens new connection for any some one file with given full SFTP URL.
func (jnt *SftpJoint) Open(fpath string) (file RFile, err error) {
	var f = SftpFile{
		jnt:  jnt,
		path: fpath,
	}
	if f.File, err = jnt.client.Open(path.Join(jnt.pwd, fpath)); err != nil {
		return
	}
	file = &f
	return
}

func (jnt *SftpJoint) Stat(fpath string) (fs.FileInfo, error) {
	return jnt.client.Stat(path.Join(jnt.pwd, fpath))
}

func (jnt *SftpJoint) ReadDir(fpath string) ([]fs.FileInfo, error) {
	return jnt.client.ReadDir(path.Join(jnt.pwd, fpath))
}

// The End.
