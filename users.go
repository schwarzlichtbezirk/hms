package hms

import (
	"encoding/xml"
	"net/http"
	"path"
	"strconv"
	"time"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

var xormUserlog *xorm.Engine

type UserStore struct {
	UID  ID_t `xorm:"pk autoincr"`
	Time Time `xorm:"created"`
}

type UaStore struct {
	UID       ID_t   `json:"uid" yaml:"uid" xml:"uid,attr"`
	UserAgent string `json:"useragent" yaml:"user-agent" xml:"useragent"` // user agent
	Lang      string `json:"lang" yaml:"lang" xml:"lang"`                 // accept language
	Addr      string `json:"addr" yaml:"addr" xml:"addr"`                 // remote address
	Time      Time   `xorm:"created"`
}

type OpenStore struct {
	UID     ID_t   `json:"uid" yaml:"uid" xml:"uid,attr"`                  // user unique ID
	AID     ID_t   `xorm:"default 0" json:"aid" yaml:"aid" xml:"aid,attr"` // access ID
	PID     ID_t   `xorm:"default 0" json:"pid" yaml:"pid" xml:"pid,attr"` // authorized profile ID
	Path    string // system path
	Latency int    // event latency, in milliseconds, or -1 if it file
	Time    Time   `xorm:"created"` // time of event rise
}

// UaMap is the set hashes of of user-agent records.
var UaMap = map[uint64]void{}

func (ust *UaStore) Hash() uint64 {
	var h = xxhash.New()
	h.Write(s2b(ust.Addr))
	h.Write(s2b(ust.UserAgent))
	return h.Sum64()
}

// UserOnline is map of last AJAX query time for each user.
var UserOnline = map[ID_t]time.Time{}

var (
	openlog = make(chan OpenStore)     // sends on folder open or any file open.
	ajaxreq = make(chan *http.Request) // sends on any ajax-call
)

// UserScanner - users scanner goroutine. Receives data from
// any API-calls to update statistics.
func UserScanner() {
	for {
		select {
		case item := <-openlog:
			go xormUserlog.InsertOne(&item)

		case r := <-ajaxreq:
			if c, err := r.Cookie("UID"); err == nil {
				var uid, _ = strconv.ParseUint(c.Value, 16, 64)
				var ust UaStore
				ust.Addr = StripPort(r.RemoteAddr)
				ust.UserAgent = r.UserAgent()
				var hv = ust.Hash()
				if _, ok := UaMap[hv]; !ok {
					ust.UID = ID_t(uid)
					if lang, ok := r.Header["Accept-Language"]; ok {
						ust.Lang = lang[0]
					}
					UaMap[hv] = void{}
					go xormUserlog.InsertOne(&ust)
				}
				UserOnline[ID_t(uid)] = time.Now()
			}

		case <-exitctx.Done():
			return
		}
	}
}

// InitUserlog inits database user log engine.
func InitUserlog() (err error) {
	if xormUserlog, err = xorm.NewEngine(xormDriverName, path.Join(CachePath, userlog)); err != nil {
		return
	}
	xormUserlog.SetMapper(names.GonicMapper{})
	xormUserlog.ShowSQL(false)

	if err = xormUserlog.Sync(&UserStore{}, &UaStore{}, &OpenStore{}); err != nil {
		return
	}
	return
}

// LoadUaMap forms content of UaMap from database on server start.
func LoadUaMap() (err error) {
	var session = xormUserlog.NewSession()
	defer session.Close()

	const limit = 256
	var offset int
	for {
		var chunk []UaStore
		if err = session.Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ust := range chunk {
			UaMap[ust.Hash()] = void{}
		}
		if limit > len(chunk) {
			break
		}
	}
	return
}

// APIHANDLER
func usrlstAPI(w http.ResponseWriter, r *http.Request) {
	type item struct { // user info
		Addr   string        `json:"addr" yaml:"addr" xml:"addr"`
		UA     uas.UserAgent `json:"ua" yaml:"ua" xml:"ua"`
		Lang   string        `json:"lang" yaml:"lang" xml:"lang"`
		Path   string        `json:"path" yaml:"path" xml:"path"`
		File   string        `json:"file" yaml:"file" xml:"file"`
		Online bool          `json:"online" yaml:"online" xml:"online"`
		AID    ID_t          `json:"accid" yaml:"accid" xml:"accid"`
		PID    ID_t          `json:"prfid" yaml:"prfid" xml:"prfid"`
	}

	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Pos int `json:"pos,omitempty" yaml:"pos,omitempty" xml:"pos,omitempty"`
		Num int `json:"num" yaml:"num" xml:"num"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Total int64  `json:"total" yaml:"total" xml:"total"`
		List  []item `json:"list" yaml:"list" xml:"list>item"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}

	var session = xormUserlog.NewSession()
	defer session.Close()

	type jstore struct {
		UserStore `xorm:"extends"`
		UaStore   `xorm:"extends"`
		Path      OpenStore `xorm:"extends"`
		File      OpenStore `xorm:"extends"`
	}

	ret.Total, _ = session.Count(&UserStore{})
	var justs []jstore
	if err = session.Distinct().Table("user_store").
		Join("INNER", "ua_store", "user_store.uid = ua_store.uid AND ua_store.time = (SELECT MIN(time) FROM ua_store WHERE uid = user_store.uid)").
		Join("INNER", "open_store t1", "user_store.uid = t1.uid AND t1.time = (SELECT MAX(time) FROM open_store WHERE uid = user_store.uid AND latency>=0)").
		Join("INNER", "open_store t2", "user_store.uid = t2.uid AND t2.time = (SELECT MAX(time) FROM open_store WHERE uid = user_store.uid AND latency=-1)").
		Limit(arg.Num, arg.Pos).Find(&justs); err != nil {
		WriteError500(w, r, err, AECusrlstusts)
		return
	}
	var now = time.Now()
	for _, rec := range justs {
		var ui = item{
			Addr:   rec.Addr,
			Lang:   rec.Lang,
			Path:   rec.Path.Path,
			File:   rec.File.Path,
			Online: now.Sub(UserOnline[rec.UaStore.UID]) < cfg.OnlineTimeout,
			AID:    rec.Path.AID,
			PID:    rec.Path.PID,
		}
		uas.ParseUserAgent(rec.UserAgent, &ui.UA)
		ret.List = append(ret.List, ui)
	}

	WriteOK(w, r, &ret)
}

// The End.
