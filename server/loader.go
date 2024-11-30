package hms

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	cfg "github.com/schwarzlichtbezirk/hms/config"
	"github.com/schwarzlichtbezirk/wpk"
	"github.com/schwarzlichtbezirk/wpk/bulk"
	"github.com/schwarzlichtbezirk/wpk/mmap"

	"github.com/gin-gonic/gin"
)

var ResFS wpk.Union // resources packages root dir.

// OpenPackage opens hms-package.
func OpenPackage() (err error) {
	for _, fname := range Cfg.WPKName {
		var t0 = time.Now()
		var fpath = JoinPath(cfg.PkgPath, fname)
		var pkg = wpk.NewPackage()
		if err = pkg.OpenFile(fpath); err != nil {
			return
		}

		var dpath string
		if pkg.IsSplitted() {
			dpath = wpk.MakeDataPath(fpath)
		} else {
			dpath = fpath
		}

		if Cfg.WPKmmap {
			pkg.Tagger, err = mmap.MakeTagger(dpath)
		} else {
			pkg.Tagger, err = bulk.MakeTagger(dpath)
		}
		PackInfo(fname, pkg, time.Since(t0))
		ResFS.List = append(ResFS.List, pkg)
	}
	return
}

// LoadTemplates is hot templates reload, during server running.
func LoadTemplates() (err error) {
	var ts, tc *template.Template
	var load = func(tb *template.Template, pattern string) {
		var tpl []string
		if tpl, err = ResFS.Glob(pattern); err != nil {
			return
		}
		for _, key := range tpl {
			var bcnt []byte
			if bcnt, err = ResFS.ReadFile(key); err != nil {
				return
			}
			var content = strings.TrimPrefix(string(bcnt), utf8bom) // remove UTF-8 format BOM header
			if _, err = tb.New(key).Parse(content); err != nil {
				return
			}
		}
	}

	ts = template.New("storage").Delims("[=[", "]=]")
	if load(ts, path.Join("tmpl", "*.html")); err != nil {
		return
	}
	if load(ts, path.Join("tmpl", "*", "*.html")); err != nil { // subfolders
		return
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(devmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(devmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}

	if tc, err = ts.Clone(); err != nil {
		return
	}
	if load(tc, path.Join(relmsuff, "*.html")); err != nil {
		return
	}
	for _, fname := range pagealias {
		var buf bytes.Buffer
		var fpath = path.Join(relmsuff, fname)
		if err = tc.ExecuteTemplate(&buf, fpath, nil); err != nil {
			return
		}
		pagecache[fpath] = buf.Bytes()
	}
	return
}

// Transaction locker, locks until handler will be done.
var handwg sync.WaitGroup

// WaitHandlers waits until all transactions will be done.
func WaitHandlers() {
	handwg.Wait()
	Log.Info("transactions completed")
}

func ApiWrap(c *gin.Context) {
	defer func() {
		if what := recover(); what != nil {
			var err error
			switch v := what.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			case fmt.Stringer:
				err = errors.New(v.String())
			default:
				err = errors.New("panic was thrown at handler")
			}
			var buf [2048]byte
			var stacklen = runtime.Stack(buf[:], false)
			var str = B2S(buf[:stacklen])
			Log.Error(str)
			Ret500(c, AEC_panic, err)
		}
	}()

	// lock before exit check
	handwg.Add(1)
	defer handwg.Done()

	var (
		cid          uint64
		uaold, uanew uint64
		isold, isnew bool
	)

	var addr, ua = c.RemoteIP(), c.Request.UserAgent()
	uanew = CalcUAID(addr, ua)

	// UAID at cookie
	if uaold, _ = GetUAID(c.Request); uaold == 0 {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:  "UAID",
			Value: strconv.FormatUint(uanew, 10),
			Path:  "/",
		})
	}

	uamux.Lock()
	if cid, isnew = UaMap[uanew]; !isnew {
		if cid, isold = UaMap[uaold]; !isold {
			maxcid++
			cid = maxcid
		}
		UaMap[uanew] = cid
		go func() {
			if _, err := XormUserlog.InsertOne(&AgentStore{
				UAID: uanew,
				CID:  cid,
				Addr: addr,
				UA:   ua,
				Lang: c.Request.Header.Get("Accept-Language"),
			}); err != nil {
				panic(err.Error())
			}
		}()
	}
	UserOnline[uanew] = time.Now()
	uamux.Unlock()

	// call the next handler
	c.Next()
}
