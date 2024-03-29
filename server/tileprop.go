package hms

import (
	"fmt"
	"strings"

	"github.com/schwarzlichtbezirk/wpk"
)

// Tiles multipliers:
//  576px: 2,  4,  6,  8, 10, 12
//  768px: 3,  6,  9, 12, 15, 18
// 1280px: 4,  8, 12, 16, 20, 24
// 1920px: 6, 12, 18, 24, 30, 36

// Tiles horizontal resolutions (tm x 24):
//  576px:  48,  96, 144, 192, 240, 288
//  768px:  72, 144, 216, 288, 360, 432
// 1280px:  96, 192, 288, 384, 480, 576
// 1920px: 144, 288, 432, 576, 720, 864

// https://go.dev/play/p/U5i5M-TfIkM

// TileProp is thumbnails properties.
type TileProp struct {
	MTmbVal Mime_t `json:"mtmb" yaml:"mtmb" xml:"mtmb"`
	MT02Val Mime_t `json:"mt02,omitempty" yaml:"mt02,omitempty" xml:"mt02,omitempty"`
	MT03Val Mime_t `json:"mt03,omitempty" yaml:"mt03,omitempty" xml:"mt03,omitempty"`
	MT04Val Mime_t `json:"mt04,omitempty" yaml:"mt04,omitempty" xml:"mt04,omitempty"`
	MT06Val Mime_t `json:"mt06,omitempty" yaml:"mt06,omitempty" xml:"mt06,omitempty"`
	MT08Val Mime_t `json:"mt08,omitempty" yaml:"mt08,omitempty" xml:"mt08,omitempty"`
	MT09Val Mime_t `json:"mt09,omitempty" yaml:"mt09,omitempty" xml:"mt09,omitempty"`
	MT10Val Mime_t `json:"mt10,omitempty" yaml:"mt10,omitempty" xml:"mt10,omitempty"`
	MT12Val Mime_t `json:"mt12,omitempty" yaml:"mt12,omitempty" xml:"mt12,omitempty"`
	MT15Val Mime_t `json:"mt15,omitempty" yaml:"mt15,omitempty" xml:"mt15,omitempty"`
	MT16Val Mime_t `json:"mt16,omitempty" yaml:"mt16,omitempty" xml:"mt16,omitempty"`
	MT18Val Mime_t `json:"mt18,omitempty" yaml:"mt18,omitempty" xml:"mt18,omitempty"`
	MT20Val Mime_t `json:"mt20,omitempty" yaml:"mt20,omitempty" xml:"mt20,omitempty"`
	MT24Val Mime_t `json:"mt24,omitempty" yaml:"mt24,omitempty" xml:"mt24,omitempty"`
	MT30Val Mime_t `json:"mt30,omitempty" yaml:"mt30,omitempty" xml:"mt30,omitempty"`
	MT36Val Mime_t `json:"mt36,omitempty" yaml:"mt36,omitempty" xml:"mt36,omitempty"`
}

const (
	htcell = 24 // horizontal tile cell length
	vtcell = 18 // vertical tile cell length
)

type TM_t int

const (
	tm0  TM_t = 0
	tm2  TM_t = 2
	tm3  TM_t = 3
	tm4  TM_t = 4
	tm6  TM_t = 6
	tm8  TM_t = 8
	tm9  TM_t = 9
	tm10 TM_t = 10
	tm12 TM_t = 12
	tm15 TM_t = 15
	tm16 TM_t = 16
	tm18 TM_t = 18
	tm20 TM_t = 20
	tm24 TM_t = 24
	tm30 TM_t = 30
	tm36 TM_t = 36
)

// CachedThumbMime returns MIME type of rendered thumbnail in package,
// or MimeNil if it not present.
func CachedThumbMime(syspath string) Mime_t {
	if ts, ok := ThumbPkg.GetTagset(syspath); ok {
		if str, ok := ts.TagStr(wpk.TIDmime); ok {
			if strings.HasPrefix(str, "image/") {
				return MimeVal[str]
			} else {
				return MimeDis
			}
		} else {
			return MimeUnk
		}
	} else {
		return MimeNil
	}
}

// CachedTileMime returns MIME type of rendered tile in package with
// given tile multiplier, or MimeNil if it not present.
func CachedTileMime(syspath string, tm TM_t) Mime_t {
	var tilepath = fmt.Sprintf("%s?%dx%d", syspath, tm*htcell, tm*vtcell)
	if ts, ok := TilesPkg.GetTagset(tilepath); ok {
		if str, ok := ts.TagStr(wpk.TIDmime); ok {
			if strings.HasPrefix(str, "image/") {
				return MimeVal[str]
			} else {
				return MimeDis
			}
		} else {
			return MimeUnk
		}
	} else {
		return MimeNil
	}
}

// Tile returns image MIME type with given tile multiplier.
func (tp *TileProp) Tile(tm TM_t) (mime Mime_t, ok bool) {
	ok = true
	switch tm {
	case tm0:
		mime = tp.MTmbVal
	case tm2:
		mime = tp.MT02Val
	case tm3:
		mime = tp.MT03Val
	case tm4:
		mime = tp.MT04Val
	case tm6:
		mime = tp.MT06Val
	case tm8:
		mime = tp.MT08Val
	case tm9:
		mime = tp.MT09Val
	case tm10:
		mime = tp.MT10Val
	case tm12:
		mime = tp.MT12Val
	case tm15:
		mime = tp.MT15Val
	case tm16:
		mime = tp.MT16Val
	case tm18:
		mime = tp.MT18Val
	case tm20:
		mime = tp.MT20Val
	case tm24:
		mime = tp.MT24Val
	case tm30:
		mime = tp.MT30Val
	case tm36:
		mime = tp.MT36Val
	default:
		mime = MimeDis
		ok = false
	}
	return
}

// SetTile updates image state to given value for tile with
// given tile multiplier.
func (tp *TileProp) SetTile(tm TM_t, mime Mime_t) (ok bool) {
	ok = true
	switch tm {
	case tm0:
		tp.MTmbVal = mime
	case tm2:
		tp.MT02Val = mime
	case tm3:
		tp.MT03Val = mime
	case tm4:
		tp.MT04Val = mime
	case tm6:
		tp.MT06Val = mime
	case tm8:
		tp.MT08Val = mime
	case tm9:
		tp.MT09Val = mime
	case tm10:
		tp.MT10Val = mime
	case tm12:
		tp.MT12Val = mime
	case tm15:
		tp.MT15Val = mime
	case tm16:
		tp.MT16Val = mime
	case tm18:
		tp.MT18Val = mime
	case tm20:
		tp.MT20Val = mime
	case tm24:
		tp.MT24Val = mime
	case tm30:
		tp.MT30Val = mime
	case tm36:
		tp.MT36Val = mime
	default:
		ok = false
	}
	return
}

// The End.
