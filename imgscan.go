package hms

import (
	"context"
	"runtime"
	"sync"
)

// Cacher provides function to perform image converting.
type Cacher interface {
	Cache()
}

// ThumbPath is thumbnail path type for cache processing.
type ThumbPath string

// Cache is Cacher implementation for ThumbPath type.
func (fpath ThumbPath) Cache() {
	var err error
	var prop interface{}
	if prop, err = propcache.Get(string(fpath)); err != nil {
		return // can not get properties
	}
	if tmb, ok := prop.(Thumber); ok {
		var tp = tmb.Tmb()
		if tp.MTmbVal != MimeNil {
			return // thumbnail already scanned
		}

		var md *MediaData
		if md, err = GetCachedThumb(string(fpath)); err != nil {
			tp.MTmbVal = MimeDis
			return
		}
		tp.MTmbVal = md.Mime
	}
}

// TilePath is tile path type for cache processing.
type TilePath struct {
	Path string
	Wdh  int
	Hgt  int
}

// Cache is Cacher implementation for TilePath type.
func (tile TilePath) Cache() {
	var err error
	var prop interface{}
	if prop, err = propcache.Get(tile.Path); err != nil {
		return // can not get properties
	}
	if tmb, ok := prop.(Thumber); ok {
		var tp = tmb.Tmb()
		var tm = TM_t(tile.Wdh / htcell)
		if mime, ok := tp.Tile(tm); ok && mime != MimeNil {
			return // thumbnail already scanned
		}

		var md *MediaData
		if md, err = GetCachedTile(tile.Path, tile.Wdh, tile.Hgt); err != nil {
			tp.SetTile(tm, MimeDis)
			return
		}
		tp.SetTile(tm, md.Mime)
	}
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
	var busy = make([]bool, thrnum)
	var free = make(chan int)
	var args = make([]chan Cacher, thrnum)
	for i := range args {
		args[i] = make(chan Cacher)
	}

	var queue []Cacher

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

// AddTmb adds PUID to queue to make thumbnails.
func (s *scanner) AddTmb(syspath string) {
	s.put <- ThumbPath(syspath)
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) RemoveTmb(syspath string) {
	s.del <- ThumbPath(syspath)
}

// AddTile adds PUID to queue to make tile with given tile multiplier.
func (s *scanner) AddTile(syspath string, tm TM_t) {
	s.put <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
}

// Remove list of PUIDs from thumbnails queue.
func (s *scanner) RemoveTile(syspath string, tm TM_t) {
	s.del <- TilePath{syspath, int(tm * htcell), int(tm * vtcell)}
}

// The End.