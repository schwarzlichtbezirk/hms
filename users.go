package hms

import (
	"crypto/md5"
	"encoding/json"
	"io/ioutil"
	"net/http"

	uas "github.com/avct/uasurfer"
)

type HistItem struct {
	PUID string `json:"puid"`
	Time int64  `json:"time"`
}

type User struct {
	Addr      string     `json:"addr" yaml:"addr"`            // remote address
	UserAgent string     `json:"useragent" yaml:"user-agent"` // user agent
	Lang      string     `json:"lang" yaml:"lang"`            // accept language
	LastAjax  int64      `json:"lastajax" yaml:"last-ajax"`   // last ajax-call UNIX-time in milliseconds
	LastPage  int64      `json:"lastpage" yaml:"last-page"`   // last page load UNIX-time in milliseconds
	IsAuth    bool       `json:"isauth" yaml:"is-auth"`       // is user authorized
	AuthID    int        `json:"authid" yaml:"auth-id"`       // authorized ID
	AccID     int        `json:"accid" yaml:"acc-id"`         // page account ID
	Paths     []HistItem `json:"paths" yaml:"paths"`          // list of opened system paths
	Files     []HistItem `json:"files" yaml:"files"`          // list of served files

	// private parsed user agent data
	ua uas.UserAgent
}

func (user *User) ParseUserAgent() {
	uas.ParseUserAgent(user.UserAgent, &user.ua)
}

type UserCache struct {
	keyuser map[string]*User
	list    []*User
}

func UserKey(addr, agent string) string {
	var h = md5.Sum([]byte(addr + agent))
	var key = idenc.EncodeToString(h[:])
	return key
}

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
	keyuser: map[string]*User{},
	list:    []*User{},
}

type userpuid struct {
	r    *http.Request
	puid string
}

type UsrMsg struct {
	r   *http.Request
	msg string
	val interface{}
}

var (
	usermsg  = make(chan UsrMsg)
	userajax = make(chan *http.Request)
	userquit = make(chan int)
)

// Users scanner goroutine. Receives data from any API-calls to update statistics.
func UserScanner() {
	for {
		select {
		case um := <-usermsg:
			var user = usercache.Get(um.r)
			user.LastAjax = UnixJSNow()
			switch um.msg {
			case "page":
				user.LastPage = user.LastAjax
				user.AccID = (um.val).(int)
			case "path":
				user.Paths = append(user.Paths, HistItem{(um.val).(string), UnixJSNow()})
			case "file":
				user.Files = append(user.Files, HistItem{(um.val).(string), UnixJSNow()})
			}

		case r := <-userajax:
			var user = usercache.Get(r)
			user.LastAjax = UnixJSNow()

		case <-userquit:
			return
		}
	}
}

// APIHANDLER
func usrlstApi(w http.ResponseWriter, r *http.Request) {
	type item struct { // user info
		Addr   string        `json:"addr"`
		UA     uas.UserAgent `json:"ua"`
		Lang   string        `json:"lang"`
		Path   string        `json:"path"`
		File   string        `json:"file"`
		Online bool          `json:"online"`
	}

	var err error
	var arg struct {
		Pos int `json:"pos"`
		Num int `json:"num"`
	}
	var ret struct {
		Total  int    `json:"total"`
		Online int    `json:"online"`
		List   []item `json:"list"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_usrlstbadreq)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_usrlstnoreq)
		return
	}

	ret.Total = len(usercache.list)
	var ot = UnixJSNow() - cfg.OnlineTimeout
	var n = 0
	for i, user := range usercache.list {
		if user.LastAjax > ot {
			ret.Online++
		}
		if i >= arg.Pos && n < arg.Num {
			var ui item
			ui.Addr = user.Addr
			ui.UA = user.ua
			ui.Lang = user.Lang
			if len(user.Paths) > 0 {
				var path, _ = pathcache.Path(user.Paths[len(user.Paths)-1].PUID)
				ui.Path = PathBase(path)
			}
			if len(user.Files) > 0 {
				var path, _ = pathcache.Path(user.Files[len(user.Files)-1].PUID)
				ui.File = PathBase(path)
			}
			ui.Online = user.LastAjax > ot
			ret.List = append(ret.List, ui)
			n++
		}
	}

	WriteOK(w, ret)
}

// The End.
