package hms

import (
	"encoding/xml"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Check whether thumbnails of pointed images prepared.
func SpiTagsCheck(c *gin.Context) {
	type store struct {
		PUID    Puid_t `xorm:"puid" json:"puid" yaml:"puid" xml:"puid,attr"`
		ExtProp `xorm:"extends" yaml:",inline"`
	}
	var err error
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>puid" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`
		List    []store  `json:"list" yaml:"list" xml:"list>tile"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tagschk_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_tagschk_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_tagschk_noacc, ErrNoAcc)
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

	RetOk(c, ret)
}

// Start to preparing thumbnails of pointed images.
func SpiTagsStart(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>tiletm" binding:"required"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tagsstart_nobind, err)
		return
	}

	for _, puid := range arg.List {
		ImgScanner.AddTags(puid)
	}

	c.Status(http.StatusOK)
}

// Break preparing thumbnails of pointed images.
func SpiTagsBreak(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []Puid_t `json:"list" yaml:"list" xml:"list>tiletm" binding:"required"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tagsbreak_nobind, err)
		return
	}

	for _, puid := range arg.List {
		ImgScanner.RemoveTags(puid)
	}

	c.Status(http.StatusOK)
}

// Check whether tiles of pointed images prepared.
func SpiTileCheck(c *gin.Context) {
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
	var ok bool
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>puid" binding:"required"`
	}
	var ret struct {
		XMLName xml.Name   `json:"-" yaml:"-" xml:"ret"`
		List    []tilemime `json:"list" yaml:"list" xml:"list>tile"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tilechk_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_tilechk_badacc, ErrNoAcc)
		return
	}
	var acc *Profile
	if acc, ok = Profiles.Get(aid); !ok {
		Ret404(c, SEC_tilechk_noacc, ErrNoAcc)
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

	RetOk(c, ret)
}

// Start to preparing tiles of pointed images.
func SpiTileStart(c *gin.Context) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>tiletm" binding:"required"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tilestart_nobind, err)
		return
	}

	for _, ttm := range arg.List {
		ImgScanner.AddTile(ttm.PUID, ttm.TM)
	}

	c.Status(http.StatusOK)
}

// Break preparing tiles of pointed images.
func SpiTileBreak(c *gin.Context) {
	type tiletm struct {
		PUID Puid_t `json:"puid" yaml:"puid" xml:"puid,attr"`
		TM   TM_t   `json:"tm,omitempty" yaml:"tm,omitempty" xml:"tm,omitempty,attr"`
	}
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		List    []tiletm `json:"list" yaml:"list" xml:"list>tiletm" binding:"required"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_tilebreak_nobind, err)
		return
	}

	for _, ttm := range arg.List {
		ImgScanner.RemoveTile(ttm.PUID, ttm.TM)
	}

	c.Status(http.StatusOK)
}

// The End.
