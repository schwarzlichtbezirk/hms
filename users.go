package hms

import (
	"encoding/xml"
	"net/http"
	"time"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash"
)

// HistItem is history item. Contains PUID of served file or
// opened directory and UNIX-time in milliseconds of start of this event.
type HistItem struct {
	PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
	Time Unix_t `json:"time" yaml:"time" xml:"time,attr"`
}

// User has vomplete information about user activity on server,
// identified by remote address and user agent.
type User struct {
	Addr      string     `json:"addr" yaml:"addr" xml:"addr"`                   // remote address
	UserAgent string     `json:"useragent" yaml:"user-agent" xml:"useragent"`   // user agent
	Lang      string     `json:"lang" yaml:"lang" xml:"lang,attr"`              // accept language
	LastAjax  Unix_t     `json:"lastajax" yaml:"last-ajax" xml:"lastajax,attr"` // last ajax-call UNIX-time in milliseconds
	LastPage  Unix_t     `json:"lastpage" yaml:"last-page" xml:"lastpage,attr"` // last page load UNIX-time in milliseconds
	IsAuth    bool       `json:"isauth" yaml:"is-auth" xml:"isauth,attr"`       // is user authorized
	AuthID    ID_t       `json:"authid" yaml:"auth-id" xml:"authid,attr"`       // authorized ID
	PrfID     ID_t       `json:"prfid" yaml:"prf-id" xml:"prfid,attr"`          // page profile ID
	Paths     []HistItem `json:"paths" yaml:"paths" xml:"paths"`                // list of opened system paths
	Files     []HistItem `json:"files" yaml:"files" xml:"files"`                // list of served files

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

var userkeyhash = xxhash.New()

// UserKey returns unique for this server session key for address
// plus user-agent, produced on fast ID_t-hash.
func UserKey(addr, agent string) ID_t {
	userkeyhash.Reset()
	userkeyhash.Write([]byte(addr + agent))
	return ID_t(userkeyhash.Sum64())
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

		case r := <-userajax:
			var user = usercache.Get(r)
			user.LastAjax = UnixJSNow()

		case <-exitctx.Done():
			return
		}
	}
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

	var session = xormEngine.NewSession()
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
			ui.Path = PathBase(fpath)
		}
		if len(user.Files) > 0 {
			var fpath, _ = PathStorePath(session, user.Files[len(user.Files)-1].PUID)
			ui.File = PathBase(fpath)
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
