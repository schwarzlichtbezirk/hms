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
	LastPage  int64      `json:"lastpage" yaml:"last-page"`   // last page load UNIX-time in milliseconds
	LastAjax  int64      `json:"lastajax" yaml:"last-ajax"`   // last ajax-call UNIX-time in milliseconds
	AID       int        `json:"aid" yaml:"aid"`              // last call account ID
	Paths     []HistItem `json:"paths" yaml:"paths"`          // list of opened system paths
	Files     []HistItem `json:"files" yaml:"files"`          // list of served files

	// private parsed user agent data
	ua uas.UserAgent
}

type UserInfo struct {
	Addr   string        `json:"addr"`
	UA     uas.UserAgent `json:"ua"`
	Lang   string        `json:"lang"`
	Path   string        `json:"path"`
	File   string        `json:"file"`
	Online bool          `json:"online"`
}

type userfilepath struct {
	r    *http.Request
	puid string
}

var usercache = map[string]*User{}
var userlist = []*User{}

var (
	userpage = make(chan *http.Request)
	userajax = make(chan *http.Request)
	userquit = make(chan int)
	userpath = make(chan userfilepath)
	userfile = make(chan userfilepath)
)

func GetUser(r *http.Request) *User {
	var addr = StripPort(r.RemoteAddr)
	var agent = r.UserAgent()
	var h = md5.Sum([]byte(addr + agent))
	var key = idenc.EncodeToString(h[:])
	var user, ok = usercache[key]
	if !ok {
		user = &User{
			Addr:      addr,
			UserAgent: agent,
		}
		uas.ParseUserAgent(agent, &user.ua)
		if lang, ok := r.Header["Accept-Language"]; ok && len(lang) > 0 {
			user.Lang = lang[0]
		}
		usercache[key] = user
		userlist = append(userlist, user)
	}
	return user
}

// Users scanner goroutine. Receives data from any API-calls to update statistics.
func UserScanner() {
	for {
		select {
		case r := <-userpage:
			var user = GetUser(r)
			user.LastPage = UnixJSNow()

		case r := <-userajax:
			var user = GetUser(r)
			user.LastAjax = UnixJSNow()

		case up := <-userpath:
			var user = GetUser(up.r)
			user.Paths = append(user.Paths, HistItem{up.puid, UnixJSNow()})

		case up := <-userfile:
			var user = GetUser(up.r)
			user.Files = append(user.Files, HistItem{up.puid, UnixJSNow()})

		case <-userquit:
			return
		}
	}
}

// APIHANDLER
func usrlstApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Pos int `json:"pos"`
		Num int `json:"num"`
	}
	var ret struct {
		Total  int        `json:"total"`
		Online int        `json:"online"`
		List   []UserInfo `json:"list"`
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

	ret.Total = len(userlist)
	var ot = UnixJSNow() - cfg.OnlineTimeout
	var n = 0
	for i, user := range userlist {
		if user.LastAjax > ot {
			ret.Online++
		}
		if i >= arg.Pos && n < arg.Num {
			var ui UserInfo
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
