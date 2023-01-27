package hms

import (
	"encoding/binary"
	"encoding/xml"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"

	uas "github.com/avct/uasurfer"
	"github.com/cespare/xxhash"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

var xormUserlog *xorm.Engine

type ClientStore struct {
	CID  ID_t `xorm:"pk autoincr"`
	Time Time `xorm:"created"`
}

type AgentStore struct {
	CID  ID_t
	Addr string // remote address
	UA   string // user agent
	Lang string // accept language
	Time Time   `xorm:"created"`
}

type OpenStore struct {
	CID     ID_t   // client unique ID
	AID     ID_t   `xorm:"default 0"` // access ID
	UID     ID_t   `xorm:"default 0"` // user profile ID
	Path    string // system path
	Latency int    // event latency, in milliseconds, or -1 if it file
	Time    Time   `xorm:"created"` // time of event rise
}

var (
	// UserOnline is map of last AJAX query time for each user.
	UserOnline = map[ID_t]time.Time{}
	// UaMap is the set hashes of of user-agent records.
	UaMap = map[uint64]void{}
	// mutex to get access to user-agent maps.
	uamux sync.Mutex
)

func (ast *AgentStore) Hash() uint64 {
	var h = xxhash.New()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(ast.CID))
	h.Write(buf[:])
	h.Write(s2b(ast.Addr))
	h.Write(s2b(ast.UA))
	return h.Sum64()
}

// GetCID extract client ID from cookie.
func GetCID(r *http.Request) (cid ID_t, err error) {
	var c *http.Cookie
	if c, err = r.Cookie("CID"); err != nil {
		return
	}
	var u64 uint64
	if u64, err = strconv.ParseUint(c.Value, 16, 64); err != nil {
		return
	}
	cid = ID_t(u64)
	return
}

// InitUserlog inits database user log engine.
func InitUserlog() (err error) {
	if xormUserlog, err = xorm.NewEngine(xormDriverName, path.Join(CachePath, userlog)); err != nil {
		return
	}
	xormUserlog.SetMapper(names.GonicMapper{})
	xormUserlog.ShowSQL(false)

	if err = xormUserlog.Sync(&ClientStore{}, &AgentStore{}, &OpenStore{}); err != nil {
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
		var chunk []AgentStore
		if err = session.Limit(limit, offset).Find(&chunk); err != nil {
			return
		}
		offset += limit
		for _, ast := range chunk {
			UaMap[ast.Hash()] = void{}
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
		UID    ID_t          `json:"usrid" yaml:"usrid" xml:"usrid"`
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
		ClientStore `xorm:"extends"`
		AgentStore  `xorm:"extends"`
		Path        OpenStore `xorm:"extends"`
		File        OpenStore `xorm:"extends"`
	}

	ret.Total, _ = session.Count(&ClientStore{})
	var justs []jstore
	if err = session.Distinct().Table("client_store").
		Join("INNER", "agent_store", "client_store.cid = agent_store.cid AND agent_store.time = (SELECT MIN(time) FROM agent_store WHERE cid = client_store.cid)").
		Join("INNER", "open_store t1", "client_store.cid = t1.cid AND t1.time = (SELECT MAX(time) FROM open_store WHERE cid = client_store.cid AND latency>=0)").
		Join("INNER", "open_store t2", "client_store.cid = t2.cid AND t2.time = (SELECT MAX(time) FROM open_store WHERE cid = client_store.cid AND latency=-1)").
		Limit(arg.Num, arg.Pos).Find(&justs); err != nil {
		WriteError500(w, r, err, AECusrlstusts)
		return
	}
	var now = time.Now()
	for _, rec := range justs {
		uamux.Lock()
		var online = now.Sub(UserOnline[rec.AgentStore.CID]) < cfg.OnlineTimeout
		uamux.Unlock()
		var ui = item{
			Addr:   rec.Addr,
			Lang:   rec.Lang,
			Path:   rec.Path.Path,
			File:   rec.File.Path,
			Online: online,
			AID:    rec.Path.AID,
			UID:    rec.Path.UID,
		}
		uas.ParseUserAgent(rec.UA, &ui.UA)
		ret.List = append(ret.List, ui)
	}

	WriteOK(w, r, &ret)
}

// The End.
