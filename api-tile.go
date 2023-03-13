package hms

import (
	"encoding/xml"
	"net/http"
)

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

		List []tiletm `json:"list" yaml:"list" xml:"list>puid"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		List []tilemime `json:"list" yaml:"list" xml:"list>tile"`
	}

	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
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

	var session = xormStorage.NewSession()
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
func tilescnstartAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		List []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECscnstartnoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECscnstartnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	for _, ttm := range arg.List {
		if syspath, ok := PathStorePath(session, ttm.PUID); ok {
			if acc.PathAccess(syspath, uid == aid) {
				ImgScanner.AddTile(syspath, ttm.TM)
			}
		}
	}

	WriteOK(w, r, nil)
}

// APIHANDLER
func tilescnbreakAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		List []tiletm `json:"list" yaml:"list" xml:"list>tiletm"`
	}

	var acc *Profile
	if acc = prflist.ByID(aid); acc == nil {
		WriteError400(w, r, ErrNoAcc, AECscnbreaknoacc)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.List) == 0 {
		WriteError400(w, r, ErrNoData, AECscnbreaknodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	for _, ttm := range arg.List {
		if syspath, ok := PathStorePath(session, ttm.PUID); ok {
			if acc.PathAccess(syspath, uid == aid) {
				ImgScanner.RemoveTile(syspath, ttm.TM)
			}
		}
	}

	WriteOK(w, r, nil)
}

// The End.
