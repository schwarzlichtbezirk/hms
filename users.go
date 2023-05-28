package hms

import (
	"encoding/xml"
	"net/http"
	"path"
	"sync"
	"time"

	. "github.com/schwarzlichtbezirk/hms/config"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

var XormUserlog *xorm.Engine

// AgentStore is storage record with user agent string and user host address.
type AgentStore struct {
	UAID ID_t      `xorm:"unique"` // user agent ID
	CID  ID_t      // client ID
	Addr string    // remote address
	UA   string    // user agent
	Lang string    // accept language
	Time time.Time `xorm:"created"`
}

// OpenStore is storage record with some opened file or opened folder.
type OpenStore struct {
	UAID    ID_t      // client ID
	AID     ID_t      `xorm:"default 0"` // access profile ID
	UID     ID_t      `xorm:"default 0"` // user profile ID
	Path    string    // system path
	Latency int       // event latency, in milliseconds, or -1 if it file
	Time    time.Time `xorm:"created"` // time of event rise
}

var (
	// UserOnline is map of last AJAX query time for each user.
	UserOnline = map[ID_t]time.Time{}
	// UaMap is the map of user agent hashes and associated client IDs.
	UaMap = map[ID_t]ID_t{}
	// current maximum client ID
	maxcid ID_t
	// mutex to get access to user-agent maps.
	uamux sync.Mutex
)

// CalcUAID calculate user agent ID by xxhash from given strings.
func CalcUAID(addr, ua string) ID_t {
	var h = xxhash.New()
	h.Write(s2b(Cfg.UaidHmacKey))
	h.Write(s2b(addr))
	h.Write(s2b(ua))
	return ID_t(h.Sum64() & 0x7fff_ffff_ffff_ffff) // clear highest bit for xorm compatibility
}

// RequestUAID calculate user agent ID from given request.
func RequestUAID(r *http.Request) ID_t {
	return CalcUAID(StripPort(r.RemoteAddr), r.UserAgent())
}

// InitUserlog inits database user log engine.
func InitUserlog() (err error) {
	if XormUserlog, err = xorm.NewEngine(Cfg.XormDriverName, path.Join(CachePath, userlog)); err != nil {
		return
	}
	XormUserlog.SetMapper(names.GonicMapper{})
	XormUserlog.ShowSQL(false)

	if err = XormUserlog.Sync(&AgentStore{}, &OpenStore{}); err != nil {
		return
	}

	var uacount, _ = XormUserlog.Count(&AgentStore{})
	Log.Infof("user agent count %d items", uacount)
	var opencount, _ = XormUserlog.Count(&OpenStore{})
	Log.Infof("resources open count %d items", opencount)
	return
}

// LoadUaMap forms content of UaMap from database on server start.
func LoadUaMap() (err error) {
	var session = XormUserlog.NewSession()
	defer session.Close()

	var u64 uint64
	if _, err = session.Table(&AgentStore{}).Select("MAX(cid)").Get(&u64); err != nil {
		return
	}
	maxcid = ID_t(u64)

	const limit = 256
	var offset int
	for {
		var chunk []AgentStore
		if err = session.Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ast := range chunk {
			UaMap[ast.UAID] = ast.CID
		}
		if limit > len(chunk) {
			break
		}
	}

	Log.Infof("clients count %d", maxcid)
	return
}

// APIHANDLER
func usrlstAPI(w http.ResponseWriter, r *http.Request) {
	type item struct { // user info
		Addr   string        `json:"addr" yaml:"addr" xml:"addr"`
		UA     uas.UserAgent `json:"ua" yaml:"ua" xml:"ua"`
		Lang   string        `json:"lang" yaml:"lang" xml:"lang"`
		Online bool          `json:"online" yaml:"online" xml:"online,attr"`
		AID    ID_t          `json:"accid" yaml:"accid" xml:"accid,attr"`
		UID    ID_t          `json:"usrid" yaml:"usrid" xml:"usrid,attr"`
		Path   string        `json:"path" yaml:"path" xml:"path"`
		File   string        `json:"file" yaml:"file" xml:"file"`
	}

	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Pos int `json:"pos,omitempty" yaml:"pos,omitempty" xml:"pos,omitempty"`
		Num int `json:"num" yaml:"num" xml:"num"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		CNum  int    `json:"cnum" yaml:"cnum" xml:"cnum"`
		UANum int    `json:"uanum" yaml:"uanum" xml:"uanum"`
		List  []item `json:"list" yaml:"list" xml:"list>item"`
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}

	var session = XormUserlog.NewSession()
	defer session.Close()

	var asts []AgentStore
	if err = XormUserlog.Limit(arg.Num, arg.Pos).Find(&asts); err != nil {
		WriteError500(w, r, err, AECusrlstasts)
		return
	}

	ret.List = make([]item, len(asts))
	var now = time.Now()
	for i, ast := range asts {
		uamux.Lock()
		var online = now.Sub(UserOnline[ast.UAID]) < Cfg.OnlineTimeout
		uamux.Unlock()
		var ui = item{
			Addr:   ast.Addr,
			Lang:   ast.Lang,
			Online: online,
		}
		uas.ParseUserAgent(ast.UA, &ui.UA)

		var is bool
		var fost, post OpenStore
		if is, err = XormUserlog.Where("uaid=? AND latency<0", ast.UAID).Desc("time").Get(&fost); err != nil {
			WriteError500(w, r, err, AECusrlstfost)
			return
		}
		if is {
			ui.File = fost.Path
			ui.AID = fost.AID
			ui.UID = fost.UID
		}
		if is, err = XormUserlog.Where("uaid=? AND latency>=0", ast.UAID).Desc("time").Get(&post); err != nil {
			WriteError500(w, r, err, AECusrlstpost)
			return
		}
		if is {
			ui.Path = post.Path
			ui.AID = post.AID
			ui.UID = post.UID
		}

		ret.List[i] = ui
	}
	uamux.Lock()
	ret.CNum = int(maxcid)
	ret.UANum = len(UaMap)
	uamux.Unlock()

	WriteOK(w, r, &ret)
}

// The End.
