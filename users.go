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
	UID       uint64 `xorm:"pk autoincr"`
	CreatedAt Time   `xorm:"created"`
}

type UaStore struct {
	UID       uint64 `json:"uid" yaml:"uid" xml:"uid,attr"`
	UserAgent string `json:"useragent" yaml:"user-agent" xml:"useragent"` // user agent
	Lang      string `json:"lang" yaml:"lang" xml:"lang"`                 // accept language
	Addr      string `json:"addr" yaml:"addr" xml:"addr"`                 // remote address
}

type OpenStore struct {
	UID     uint64 `json:"uid" yaml:"uid" xml:"uid,attr"`
	AID     uint64
	Path    string
	Time    Time
	Latency int
}

// UaMap is the set hashes of of user-agent records.
var UaMap = map[uint64]void{}

func (ust *UaStore) Hash() uint64 {
	var h = xxhash.New()
	h.Write(s2b(ust.Addr))
	h.Write(s2b(ust.UserAgent))
	return h.Sum64()
}

// HistItem is history item. Contains PUID of served file or
// opened directory and UNIX-time in milliseconds of start of this event.
type HistItem struct {
	PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
	Time Unix_t `xorm:"DateTime" json:"time" yaml:"time" xml:"time,attr"`
}

// User has vomplete information about user activity on server,
// identified by remote address and user agent.
type User struct {
	Addr      string     `json:"addr" yaml:"addr" xml:"addr"`                                   // remote address
	UserAgent string     `json:"useragent" yaml:"user-agent" xml:"useragent"`                   // user agent
	Lang      string     `json:"lang" yaml:"lang" xml:"lang,attr"`                              // accept language
	LastAjax  Unix_t     `xorm:"DateTime" json:"lastajax" yaml:"last-ajax" xml:"lastajax,attr"` // last ajax-call UNIX-time in milliseconds
	LastPage  Unix_t     `xorm:"DateTime" json:"lastpage" yaml:"last-page" xml:"lastpage,attr"` // last page load UNIX-time in milliseconds
	IsAuth    bool       `json:"isauth" yaml:"is-auth" xml:"isauth,attr"`                       // is user authorized
	AuthID    ID_t       `json:"authid" yaml:"auth-id" xml:"authid,attr"`                       // authorized ID
	PrfID     ID_t       `json:"prfid" yaml:"prf-id" xml:"prfid,attr"`                          // page profile ID
	Paths     []HistItem `json:"paths" yaml:"paths" xml:"paths"`                                // list of opened system paths
	Files     []HistItem `json:"files" yaml:"files" xml:"files"`                                // list of served files

	// private parsed user agent data
	ua uas.UserAgent
}

// ParseUserAgent parse and setup private data with structured user agent representation.
func (user *User) ParseUserAgent() {
	uas.ParseUserAgent(user.UserAgent, &user.ua)
}

// UserMap is map with users with ID_t-keys produced as hash of address plus user-agent.
type UserMap = map[ID_t]*User

// UserCache - users cache, ordered list and map.
type UserCache struct {
	keyuser UserMap
	list    []*User
}

// UserKey returns unique for this server session key for address
// plus user-agent, produced on fast uint64-hash.
func UserKey(addr, agent string) ID_t {
	var h = xxhash.New()
	h.Reset()
	h.Write([]byte(addr + agent))
	return ID_t(h.Sum64())
}

// Get returns User structure depending on http-request,
// identified by remote address and user agent.
func (uc *UserCache) Get(r *http.Request) *User {
	var addr = StripPort(r.RemoteAddr)
	var agent = r.UserAgent()
	var key = UserKey(addr, agent)
	var user, ok = uc.keyuser[key]
	if !ok {
		user = &User{
			Addr:      addr,
			UserAgent: agent,
		}
		user.ParseUserAgent()
		if lang, ok := r.Header["Accept-Language"]; ok && len(lang) > 0 {
			user.Lang = lang[0]
		}
		uc.keyuser[key] = user
		uc.list = append(uc.list, user)
	}
	return user
}

var usercache = UserCache{
	keyuser: UserMap{},
	list:    []*User{},
}

// UsrMsg is user message. Contains some chunk of data changes in user structure.
type UsrMsg struct {
	r   *http.Request
	msg string
	val any
}

var (
	openlog = make(chan OpenStore)     // sends on folder open or any file open.
	ajaxreq = make(chan *http.Request) // sends on any ajax-call
	usermsg = make(chan UsrMsg)        // message with some data of user-change
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
					ust.UID = uid
					if lang, ok := r.Header["Accept-Language"]; ok {
						ust.Lang = lang[0]
					}
					UaMap[hv] = void{}
					go xormUserlog.InsertOne(&ust)
				}
			}

			var user = usercache.Get(r)
			user.LastAjax = UnixJSNow()

		case um := <-usermsg:
			var user = usercache.Get(um.r)
			user.LastAjax = UnixJSNow()
			switch um.msg {
			case "auth":
				var aid = (um.val).(ID_t)
				if aid > 0 {
					user.IsAuth = true
					user.AuthID = aid
				} else {
					user.IsAuth = false
				}
			case "page":
				user.LastPage = user.LastAjax
				user.PrfID = (um.val).(ID_t)
			case "path":
				user.Paths = append(user.Paths, HistItem{(um.val).(Puid_t), user.LastAjax})
			case "file":
				user.Files = append(user.Files, HistItem{(um.val).(Puid_t), user.LastAjax})
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
		IsAuth bool          `json:"isauth" yaml:"isauth" xml:"isauth"`
		AuthID ID_t          `json:"authid" yaml:"authid" xml:"authid"`
		PrfID  ID_t          `json:"prfid" yaml:"prfid" xml:"prfid"`
	}

	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Pos int `json:"pos,omitempty" yaml:"pos,omitempty" xml:"pos,omitempty"`
		Num int `json:"num" yaml:"num" xml:"num"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Total int    `json:"total" yaml:"total" xml:"total"`
		List  []item `json:"list" yaml:"list" xml:"list>item"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	ret.Total = len(usercache.list)
	var ot = UnixJSNow() - Unix_t(cfg.OnlineTimeout/time.Millisecond)
	for i := arg.Pos; i < arg.Pos+arg.Num && i < len(usercache.list); i++ {
		var user = usercache.list[i]
		var ui item
		ui.Addr = user.Addr
		ui.UA = user.ua
		ui.Lang = user.Lang
		if len(user.Paths) > 0 {
			var fpath, _ = PathStorePath(session, user.Paths[len(user.Paths)-1].PUID)
			ui.Path = path.Base(fpath)
		}
		if len(user.Files) > 0 {
			var fpath, _ = PathStorePath(session, user.Files[len(user.Files)-1].PUID)
			ui.File = path.Base(fpath)
		}
		ui.Online = user.LastAjax > ot
		ui.IsAuth = user.IsAuth
		ui.AuthID = user.AuthID
		ui.PrfID = user.PrfID
		ret.List = append(ret.List, ui)
	}

	WriteOK(w, r, &ret)
}

// The End.
