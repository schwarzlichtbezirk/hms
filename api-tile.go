package hms

import (
	"encoding/xml"
	"net/http"
)

// APIHANDLER
func extchkAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type store struct {
		PUID    Puid_t `xorm:"puid" json:"puid" yaml:"puid" xml:"puid,attr"`
		ExtProp `xorm:"extends" yaml:",inline"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`
		List    []store  `json:"list" yaml:"list" xml:"list>tile"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtagschknoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtagschknodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	ret.List = make([]store, len(arg.List))
	for i, puid := range arg.List {
		var xp ExtProp
		xp.ETmb = MimeDis // disable if no access
		if syspath, ok := PathStorePath(session, puid); ok {
			if acc.PathAccess(syspath, uid == aid) {
				if xp, ok = extcache.Peek(puid); !ok {
					xp.ETmb = MimeNil // not cached yet
				}
			}
		}
		ret.List[i] = store{puid, xp}
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func extstartAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtagsstartnoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtagsstartnodata)
		return
	}

	for _, puid := range arg.List {
		ImgScanner.AddTags(puid)
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func extbreakAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtagsbreaknoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtagsbreaknodata)
		return
	}

	for _, puid := range arg.List {
		ImgScanner.RemoveTags(puid)
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func tilechkAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm" yaml:"tm" xml:"tm,omitempty,attr"`
	}
	type tilemime struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm" yaml:"tm" xml:"tm,omitempty,attr"`
		Mime Mime_t `json:"mime,omitempty" yaml:"mime,omitempty" xml:"mime,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name   `json:"-" yaml:"-" xml:"ret"`
		List    []tilemime `json:"list" yaml:"list" xml:"list>tile"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtilechknoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtilechknodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	ret.List = make([]tilemime, len(arg.List))
	for i, ttm := range arg.List {
		var mime = MimeDis // disable if no access
		if syspath, ok := PathStorePath(session, ttm.PUID); ok {
			if acc.PathAccess(syspath, uid == aid) {
				if tp, ok := tilecache.Peek(ttm.PUID); ok {
					mime, _ = tp.Tile(ttm.TM)
				} else {
					mime = MimeNil // not cached yet
				}
			}
		}
		ret.List[i].PUID, ret.List[i].TM, ret.List[i].Mime = ttm.PUID, ttm.TM, mime
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func tilestartAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtilestartnoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtilestartnodata)
		return
	}

	for _, ttm := range arg.List {
		ImgScanner.AddTile(ttm.PUID, ttm.TM)
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func tilebreakAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = ProfileByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECtilebreaknoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECtilebreaknodata)
		return
	}

	for _, ttm := range arg.List {
		ImgScanner.RemoveTile(ttm.PUID, ttm.TM)
	}

	WriteOK(w, r, nil)
}

// The End.
