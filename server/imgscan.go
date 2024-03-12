package hms

import (
	"context"
	"path"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
)

// Cacher provides function to perform image converting.
type Cacher interface {
	Cache()
}

// EmbedPath is path to get embedded JPEG thumbnail.
type EmbedPath Puid_t

func (puid EmbedPath) Cache() {
	var session = XormStorage.NewSession()
	defer session.Close()

	var fpath, _ = PathStorePath(session, Puid_t(puid))

	var buf StoreBuf
	buf.Init(1) // flush on every push

	if _, _, err := TagsExtract(fpath, session, &buf, &ExtStat{}, true); err != nil {
		Log.Warnf("etmb: %s, error %v", path.Base(fpath), err)
	}
}

// ThumbPath is thumbnail path type for cache processing.
type ThumbPath Puid_t

// Cache is Cacher implementation for ThumbPath type.
func (puid ThumbPath) Cache() {
	var session = XormStorage.NewSession()
	defer session.Close()

	var fpath, _ = PathStorePath(session, Puid_t(puid))
	var err error
	var md MediaData
	if md, err = CacheThumb(session, fpath); err != nil {
		md.Mime = MimeDis
		Log.Warnf("mtmb: %s, error %v", path.Base(fpath), err)
	}

	var tp, _ = tilecache.Peek(Puid_t(puid))
	tp.SetTile(tm0, md.Mime)
	tilecache.Poke(Puid_t(puid), tp)
}

// TilePath is tile path type for cache processing.
type TilePath struct {
	Puid Puid_t
	Wdh  int
	Hgt  int
}

// Cache is Cacher implementation for TilePath type.
func (tile TilePath) Cache() {
	var session = XormStorage.NewSession()
	defer session.Close()

	var fpath, _ = PathStorePath(session, tile.Puid)
	var err error
	var md MediaData
	if md, err = CacheTile(session, fpath, tile.Wdh, tile.Hgt); err != nil {
		md.Mime = MimeDis
		Log.Warnf("tile%dx%d: %s, error %v", tile.Wdh, tile.Hgt, path.Base(fpath), err)
	}

	var tp, _ = tilecache.Peek(tile.Puid)
	var tm = TM_t(tile.Wdh / htcell)
	tp.SetTile(tm, md.Mime)
	tilecache.Poke(tile.Puid, tp)
}

// GetScanThreadsNum returns number of scanning threads
// depended of settings and developer mode.
func GetScanThreadsNum() int {
	var thrnum = Cfg.ScanThreadsNum
	if thrnum == 0 {
		thrnum = runtime.GOMAXPROCS(0)
	}
	if cfg.DevMode { // only one thread under the debugger
		thrnum = 1
	}
	return thrnum
}

// ImgScanner is singleton for thumbnails producing
// with single queue to prevent overload.
var ImgScanner scanner

type scanner struct {
	put    chan Cacher
	del    chan Cacher
	cancel context.CancelFunc
	fin    context.Context
}

// Scan is goroutine for thumbnails scanning.
func (s *scanner) Scan() {
	s.put = make(chan Cacher)
	s.del = make(chan Cacher)
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())
	var cancel context.CancelFunc
	s.fin, cancel = context.WithCancel(context.Background())
	defer func() {
		s.fin, s.cancel = nil, nil
		cancel()
	}()

	var thrnum = GetScanThreadsNum()
	var busy = make([]bool, thrnum)
	var free = make(chan int)
	var args = make([]chan Cacher, thrnum)
	for i := range args {
		args[i] = make(chan Cacher)
	}

	var queue []Cacher
	var issync uint32 // prevents a series of calls

	var wg sync.WaitGroup
	wg.Add(thrnum)
	for i := 0; i < thrnum; i++ {
		var i = i // localize
		go func() {
			defer wg.Done()
			for {
				select {
				case arg := <-args[i]:
					arg.Cache()
					free <- i
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	func() {
		for {
		selector:
			select {
			case arg := <-s.put:
				var found = false
				for i, b := range busy {
					if !b {
						busy[i] = true
						args[i] <- arg
						found = true
						break
					}
				}
				if !found {
					queue = append(queue, arg)
				}
			case arg := <-s.del:
				for i, val := range queue {
					if arg == val {
						queue = append(queue[:i], queue[i+1:]...)
						break
					}
				}
			case i := <-free:
				if len(queue) > 0 {
					var arg = queue[0]
					queue = queue[1:]
					busy[i] = true
					args[i] <- arg
				} else {
					busy[i] = false
					for _, b := range busy {
						if b {
							break selector
						}
					}
					if atomic.LoadUint32(&issync) == 0 {
						atomic.StoreUint32(&issync, 1)
						go func() {
							defer atomic.StoreUint32(&issync, 0)
							time.Sleep(500 * time.Millisecond)
							// sync file tags tables of caches
							if err := ThumbPkg.Sync(); err != nil {
								Log.Error(err)
							}
							if err := TilesPkg.Sync(); err != nil {
								Log.Error(err)
							}
							Log.Info("caches synced")
						}()
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()
}

// Stop makes the break to scanning process and returns context
// that indicates graceful scanning end.
func (s *scanner) Stop() (ctx context.Context) {
	ctx = s.fin
	if s.cancel != nil {
		s.cancel()
	}
	return
}

// AddTags adds system path to queue to extract embedded thumbnail and tags.
func (s *scanner) AddTags(puid Puid_t) {
	s.put <- EmbedPath(puid)
}

// RemoveTags removes system path for embedded thumbnail from queue.
func (s *scanner) RemoveTags(puid Puid_t) {
	s.del <- EmbedPath(puid)
}

// AddTile adds system path to queue to render thumbnail from image
// source (on tm == tm0), or to render tile with given tile multiplier
// (on any other tm case).
func (s *scanner) AddTile(puid Puid_t, tm TM_t) {
	if tm == tm0 {
		s.put <- ThumbPath(puid)
	} else {
		s.put <- TilePath{puid, int(tm * htcell), int(tm * vtcell)}
	}
}

// RemoveTile removes system path for thumbnail render (on tm == tm0),
// or for tile render from queue (on any other tm case).
func (s *scanner) RemoveTile(puid Puid_t, tm TM_t) {
	if tm == tm0 {
		s.del <- ThumbPath(puid)
	} else {
		s.del <- TilePath{puid, int(tm * htcell), int(tm * vtcell)}
	}
}

// The End.
