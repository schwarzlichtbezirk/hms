package hms

import (
	"encoding/xml"
	"sync"
	"time"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash/v2"
	"github.com/gin-gonic/gin"
	"xorm.io/xorm"
)

var XormUserlog *xorm.Engine

// AgentStore is storage record with user agent string and user host address.
type AgentStore struct {
	UAID uint64    `xorm:"unique"` // user agent ID
	CID  uint64    // client ID
	Addr string    // remote address
	UA   string    // user agent
	Lang string    // accept language
	Time time.Time `xorm:"created"`
}

// OpenStore is storage record with some opened file or opened folder.
type OpenStore struct {
	UAID    uint64    // client ID
	AID     uint64    `xorm:"default 0"` // access profile ID
	UID     uint64    `xorm:"default 0"` // user profile ID
	Path    string    // system path
	Latency int       // event latency, in milliseconds, or -1 if it file
	Time    time.Time `xorm:"created"` // time of event rise
}

var (
	// UserOnline is map of last AJAX query time for each user.
	UserOnline = map[uint64]time.Time{}
	// UaMap is the map of user agent hashes and associated client IDs.
	UaMap = map[uint64]uint64{}
	// current maximum client ID
	maxcid uint64
	// mutex to get access to user-agent maps.
	uamux sync.Mutex
)

// CalcUAID calculate user agent ID by xxhash from given strings.
func CalcUAID(addr, ua string) uint64 {
	var h = xxhash.New()
	h.Write(S2B(Cfg.UaidHmacKey))
	h.Write(S2B(addr))
	h.Write(S2B(ua))
	return h.Sum64() & 0x7fff_ffff_ffff_ffff // clear highest bit for xorm compatibility
}

// RequestUAID calculate user agent ID from given request.
func RequestUAID(c *gin.Context) uint64 {
	return CalcUAID(c.RemoteIP(), c.Request.UserAgent())
}

// LoadUaMap forms content of UaMap from database on server start.
func LoadUaMap() (err error) {
	var session = XormUserlog.NewSession()
	defer session.Close()

	if _, err = session.Table(&AgentStore{}).Select("MAX(cid)").Get(&maxcid); err != nil {
		return
	}

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
func SpiUserList(c *gin.Context) {
	type item struct { // user info
		Addr   string        `json:"addr" yaml:"addr" xml:"addr"`
		UA     uas.UserAgent `json:"ua" yaml:"ua" xml:"ua"`
		Lang   string        `json:"lang" yaml:"lang" xml:"lang"`
		Online bool          `json:"online" yaml:"online" xml:"online,attr"`
		AID    uint64        `json:"accid" yaml:"accid" xml:"accid,attr"`
		UID    uint64        `json:"usrid" yaml:"usrid" xml:"usrid,attr"`
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
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, AEC_usrlst_nobind, err)
		return
	}

	var session = XormUserlog.NewSession()
	defer session.Close()

	var asts []AgentStore
	if err = XormUserlog.Limit(arg.Num, arg.Pos).Find(&asts); err != nil {
		Ret500(c, AEC_usrlst_asts, err)
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
			Ret500(c, AEC_usrlst_fost, err)
			return
		}
		if is {
			ui.File = fost.Path
			ui.AID = fost.AID
			ui.UID = fost.UID
		}
		if is, err = XormUserlog.Where("uaid=? AND latency>=0", ast.UAID).Desc("time").Get(&post); err != nil {
			Ret500(c, AEC_usrlst_post, err)
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

	RetOk(c, ret)
}

// The End.
