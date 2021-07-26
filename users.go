package hms

import (
	"net/http"
	"time"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash"
)

// HistItem is history item. Contains PUID of served file or
// opened directory and UNIX-time in milliseconds of start of this event.
type HistItem struct {
	PUID string `json:"puid"`
	Time int64  `json:"time"`
}

// User has vomplete information about user activity on server,
// identified by remote address and user agent.
type User struct {
	Addr      string     `json:"addr" yaml:"addr"`            // remote address
	UserAgent string     `json:"useragent" yaml:"user-agent"` // user agent
	Lang      string     `json:"lang" yaml:"lang"`            // accept language
	LastAjax  int64      `json:"lastajax" yaml:"last-ajax"`   // last ajax-call UNIX-time in milliseconds
	LastPage  int64      `json:"lastpage" yaml:"last-page"`   // last page load UNIX-time in milliseconds
	IsAuth    bool       `json:"isauth" yaml:"is-auth"`       // is user authorized
	AuthID    int        `json:"authid" yaml:"auth-id"`       // authorized ID
	PrfID     int        `json:"prfid" yaml:"prf-id"`         // page profile ID
	Paths     []HistItem `json:"paths" yaml:"paths"`          // list of opened system paths
	Files     []HistItem `json:"files" yaml:"files"`          // list of served files

	// private parsed user agent data
	ua uas.UserAgent
}

// ParseUserAgent parse and setup private data with structured user agent representation.
func (user *User) ParseUserAgent() {
	uas.ParseUserAgent(user.UserAgent, &user.ua)
}

// UserMap is map with users with uint64-keys produced as hash of address plus user-agent.
type UserMap = map[uint64]*User

// UserCache - users cache, ordered list and map.
type UserCache struct {
	keyuser UserMap
	list    []*User
}

var userkeyhash = xxhash.New()

// UserKey returns unique for this server session key for address
// plus user-agent, produced on fast uint64-hash.
func UserKey(addr, agent string) uint64 {
	userkeyhash.Reset()
	userkeyhash.Write([]byte(addr + agent))
	return userkeyhash.Sum64()
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
	val interface{}
}

var (
	usermsg  = make(chan UsrMsg)        // message with some data of user-change
	userajax = make(chan *http.Request) // sends on any ajax-call
)

// UserScanner - users scanner goroutine. Receives data from
// any API-calls to update statistics.
func UserScanner() {
	for {
		select {
		case um := <-usermsg:
			var user = usercache.Get(um.r)
			user.LastAjax = UnixJSNow()
			switch um.msg {
			case "auth":
				var aid = (um.val).(int)
				if aid > 0 {
					user.IsAuth = true
					user.AuthID = aid
				} else {
					user.IsAuth = false
				}
			case "page":
				user.LastPage = user.LastAjax
				user.PrfID = (um.val).(int)
			case "path":
				user.Paths = append(user.Paths, HistItem{(um.val).(string), user.LastAjax})
			case "file":
				user.Files = append(user.Files, HistItem{(um.val).(string), user.LastAjax})
			}

		case r := <-userajax:
			var user = usercache.Get(r)
			user.LastAjax = UnixJSNow()

		case <-exitchan:
			return
		}
	}
}

// APIHANDLER
func usrlstAPI(w http.ResponseWriter, r *http.Request) {
	type item struct { // user info
		Addr   string        `json:"addr"`
		UA     uas.UserAgent `json:"ua"`
		Lang   string        `json:"lang"`
		Path   string        `json:"path"`
		File   string        `json:"file"`
		Online bool          `json:"online"`
		IsAuth bool          `json:"isauth"`
		AuthID int           `json:"authid"`
		PrfID  int           `json:"prfid"`
	}

	var err error
	var arg struct {
		Pos int `json:"pos"`
		Num int `json:"num"`
	}
	var ret struct {
		Total int    `json:"total"`
		List  []item `json:"list"`
	}

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}

	ret.Total = len(usercache.list)
	var ot = UnixJSNow() - int64(cfg.OnlineTimeout/time.Millisecond)
	for i := arg.Pos; i < arg.Pos+arg.Num && i < len(usercache.list); i++ {
		var user = usercache.list[i]
		var ui item
		ui.Addr = user.Addr
		ui.UA = user.ua
		ui.Lang = user.Lang
		if len(user.Paths) > 0 {
			var fpath, _ = pathcache.Path(user.Paths[len(user.Paths)-1].PUID)
			ui.Path = PathBase(fpath)
		}
		if len(user.Files) > 0 {
			var fpath, _ = pathcache.Path(user.Files[len(user.Files)-1].PUID)
			ui.File = PathBase(fpath)
		}
		ui.Online = user.LastAjax > ot
		ui.IsAuth = user.IsAuth
		ui.AuthID = user.AuthID
		ui.PrfID = user.PrfID
		ret.List = append(ret.List, ui)
	}

	WriteOK(w, ret)
}

// The End.
