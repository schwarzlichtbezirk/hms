package joint

import (
	"errors"
	"io"
	"io/fs"
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
)

var (
	ErrNotFound = errors.New("resource not found")
	ErrUnexpDir = errors.New("unexpected directory instead the file")
	ErrNotIso   = errors.New("filesystem is not ISO9660")
)

// Joint describes interface with joint to some file system provider.
type Joint interface {
	Make(urladdr string) error             // establish connection to file system provider
	Cleanup() error                        // close connection to file system provider
	Key() string                           // key path to resource
	Busy() bool                            // file is opened
	fs.FS                                  // open file with local file path
	io.Closer                              // close local file
	Info(string) (fs.FileInfo, error)      // returns file state pointed by local file path
	ReadDir(string) ([]fs.FileInfo, error) // read directory pointed by local file path
	RFile
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

// The End.
