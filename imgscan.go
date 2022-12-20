package hms

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Cacher provides function to perform image converting.
type Cacher interface {
	Cache()
}

// EmbedPath is path to get embedded JPEG thumbnail.
type EmbedPath string

func (fpath EmbedPath) Cache() {
	var session = xormEngine.NewSession()
	defer session.Close()

	var puid = PathStoreCache(session, string(fpath))
	var err error
	var md MediaData
	if md, err = ExtractThmub(session, string(fpath)); err != nil {
		md.Mime = MimeDis
	}

	var tp, ok = tilecache.Peek(puid)
	if !ok {
		tp = &TileProp{}
	}
	tp.SetTile(tme, md.Mime)
	tilecache.Push(puid, tp)
}

// ThumbPath is thumbnail path type for cache processing.
type ThumbPath string

// Cache is Cacher implementation for ThumbPath type.
func (fpath ThumbPath) Cache() {
	var session = xormEngine.NewSession()
	defer session.Close()

	var puid = PathStoreCache(session, string(fpath))
	var err error
	var md MediaData
	if md, err = CacheThumb(session, string(fpath)); err != nil {
		md.Mime = MimeDis
	}

	var tp, ok = tilecache.Peek(puid)
	if !ok {
		tp = &TileProp{}
	}
	tp.SetTile(tm0, md.Mime)
	tilecache.Push(puid, tp)
}

// TilePath is tile path type for cache processing.
type TilePath struct {
	Path string
	Wdh  int
	Hgt  int
}

// Cache is Cacher implementation for TilePath type.
func (tile TilePath) Cache() {
	var session = xormEngine.NewSession()
	defer session.Close()

	var puid = PathStoreCache(session, tile.Path)
	var err error
	var md MediaData
	if md, err = CacheTile(session, tile.Path, tile.Wdh, tile.Hgt); err != nil {
		md.Mime = MimeDis
	}

	var tp, ok = tilecache.Peek(puid)
	if !ok {
		tp = &TileProp{}
	}
	var tm = TM_t(tile.Wdh / htcell)
	tp.SetTile(tm, md.Mime)
	tilecache.Push(puid, tp)
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

	var thrnum = cfg.ScanThreadsNum
	if thrnum == 0 {
		thrnum = runtime.GOMAXPROCS(0)
	}
	if devmode { // only one thread under the debugger
		thrnum = 1
	}
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
							if err := thumbpkg.Sync(); err != nil {
								Log.Infoln(err)
							}
							if err := tilespkg.Sync(); err != nil {
								Log.Infoln(err)
							}
							Log.Infoln("caches synced")
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

// AddTile adds system path to queue to extract embedded thumbnail
// (on tm == tme), or to render thumbnail from image source (on tm == tm0),
// or to render tile with given tile multiplier (on any other tm case).
func (s *scanner) AddTile(syspath string, tm TM_t) {
	if tm == tme {
		s.put <- EmbedPath(syspath)
	} else if tm == tm0 {
		s.put <- ThumbPath(syspath)
	} else {
		s.put <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
	}
}

// RemoveTmb removes system path for embedded thumbnail from queue
// (on tm == tme), or for thumbnail render (on tm == tm0), or for
// tile render from queue (on any other tm case).
func (s *scanner) RemoveTile(syspath string, tm TM_t) {
	if tm == tme {
		s.del <- EmbedPath(syspath)
	} else if tm == tm0 {
		s.del <- ThumbPath(syspath)
	} else {
		s.del <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
	}
}

// The End.
