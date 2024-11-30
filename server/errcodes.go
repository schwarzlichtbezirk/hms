package hms

import "errors"

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AEC_null = iota
	AEC_badenc
	AEC_panic

	// authorization

	AEC_auth_absent
	AEC_auth_scheme
	AEC_basic_decode
	AEC_basic_noacc
	AEC_basic_deny
	AEC_token_noacc
	AEC_token_malform
	AEC_token_notsign
	AEC_token_badclaims
	AEC_token_expired
	AEC_token_notyet
	AEC_token_issuer
	AEC_token_error
	AEC_token_badaid
	AEC_param_noacc

	// 404

	AEC_nourl

	// 405

	AEC_nomethod

	// auth/signin

	AEC_signin_nobind
	AEC_signin_nosecret
	AEC_signin_smallsec
	AEC_signin_nouser
	AEC_signin_denypass
	AEC_signin_sigtime
	AEC_signin_timeout
	AEC_signin_hs256
	AEC_signin_denyhash

	// page

	AEC_page_absent

	// file

	AEC_media_badacc
	AEC_media_noacc
	AEC_media_badmedia
	AEC_media_badhd
	AEC_media_badpath
	AEC_media_hidden
	AEC_media_access
	AEC_media_hdgone
	AEC_media_hdfail
	AEC_media_hdnocnt
	AEC_media_medgone
	AEC_media_medfail
	AEC_media_mednocnt
	AEC_media_filegone
	AEC_media_fileopen

	// etmb

	AEC_etmb_badacc
	AEC_etmb_noacc
	AEC_etmb_nopuid
	AEC_etmb_nopath
	AEC_etmb_hidden
	AEC_etmb_access
	AEC_etmb_badcnt
	AEC_etmb_notmb

	// mtmb

	AEC_mtmb_badacc
	AEC_mtmb_noacc
	AEC_mtmb_nopuid
	AEC_mtmb_nopath
	AEC_mtmb_hidden
	AEC_mtmb_access
	AEC_mtmb_badcnt
	AEC_mtmb_absent

	// tile

	AEC_tile_badacc
	AEC_tile_noacc
	AEC_tile_nopuid
	AEC_tile_twodim
	AEC_tile_badwdh
	AEC_tile_badhgt
	AEC_tile_zero
	AEC_tile_nopath
	AEC_tile_hidden
	AEC_tile_access
	AEC_tile_badcnt
	AEC_tile_absent

	// reload

	AEC_reload_load
	AEC_reload_tmpl

	// stat/getlog

	AEC_getlog_nobind

	// res/folder

	AEC_folder_nobind
	AEC_folder_badacc
	AEC_folder_noacc
	AEC_folder_badpath
	AEC_folder_hidden
	AEC_folder_access
	AEC_folder_stat
	AEC_folder_noshr
	AEC_folder_home
	AEC_folder_drives
	AEC_folder_remote
	AEC_folder_shares
	AEC_folder_media
	AEC_folder_map
	AEC_folder_nocat
	AEC_folder_absent
	AEC_folder_fail
	AEC_folder_open
	AEC_folder_m3u
	AEC_folder_wpl
	AEC_folder_pls
	AEC_folder_asx
	AEC_folder_xspf
	AEC_folder_format
	AEC_folder_tracks

	// res/tags
	AEC_tags_nobind
	AEC_tags_badacc
	AEC_tags_noacc
	AEC_tags_badpath
	AEC_tags_hidden
	AEC_tags_access
	AEC_tags_extract

	// res/ispath

	AEC_ispath_nobind
	AEC_ispath_badacc
	AEC_ispath_noacc
	AEC_ispath_deny

	// tags/chk

	AEC_tagschk_nobind
	AEC_tagschk_badacc
	AEC_tagschk_noacc

	// tags/start

	AEC_tagsstart_nobind

	// tags/break

	AEC_tagsbreak_nobind

	// tile/chk

	AEC_tilechk_nobind
	AEC_tilechk_badacc
	AEC_tilechk_noacc

	// tile/start

	AEC_tilestart_nobind

	// tile/break

	AEC_tilebreak_nobind

	// drive/add

	AEC_drvadd_nobind
	AEC_drvadd_badacc
	AEC_drvadd_noacc
	AEC_drvadd_deny
	AEC_drvadd_badpath
	AEC_drvadd_miss
	AEC_drvadd_hidden

	// drive/del

	AEC_drvdel_nobind
	AEC_drvdel_badacc
	AEC_drvdel_noacc
	AEC_drvdel_deny
	AEC_drvdel_nopath

	// cloud/add

	AEC_cldadd_nobind
	AEC_cldadd_badacc
	AEC_cldadd_noacc
	AEC_cldadd_deny
	AEC_cldadd_badhost
	AEC_cldadd_ftpdial
	AEC_cldadd_ftpcred
	AEC_cldadd_ftproot
	AEC_cldadd_sftpdial
	AEC_cldadd_sftpcli
	AEC_cldadd_sftppwd
	AEC_cldadd_sftproot
	AEC_cldadd_davdial

	// cloud/del

	AEC_clddel_nobind
	AEC_clddel_badacc
	AEC_clddel_noacc
	AEC_clddel_deny
	AEC_clddel_nopath

	// share/add

	AEC_shradd_nobind
	AEC_shradd_badacc
	AEC_shradd_noacc
	AEC_shradd_deny
	AEC_shradd_nopath
	AEC_shradd_access

	// share/del

	AEC_shrdel_nobind
	AEC_shrdel_badacc
	AEC_shrdel_noacc
	AEC_shrdel_deny
	AEC_shrdel_nopath
	AEC_shrdel_access

	// edit/copy

	AEC_edtcopy_nobind
	AEC_edtcopy_badacc
	AEC_edtcopy_noacc
	AEC_edtcopy_deny
	AEC_edtcopy_nopath
	AEC_edtcopy_nodest
	AEC_edtcopy_over
	AEC_edtcopy_opsrc
	AEC_edtcopy_statsrc
	AEC_edtcopy_mkdir
	AEC_edtcopy_rd
	AEC_edtcopy_opdst
	AEC_edtcopy_copy
	AEC_edtcopy_statfile

	// edit/rename

	AEC_edtren_nobind
	AEC_edtren_badacc
	AEC_edtren_noacc
	AEC_edtren_deny
	AEC_edtren_nopath
	AEC_edtren_nodest
	AEC_edtren_over
	AEC_edtren_move
	AEC_edtren_stat

	// edit/del

	AEC_edtdel_nobind
	AEC_edtdel_badacc
	AEC_edtdel_noacc
	AEC_edtdel_deny
	AEC_edtdel_nopath
	AEC_edtdel_remove

	// gps/range

	AEC_gpsrange_nobind
	AEC_gpsrange_badacc
	AEC_gpsrange_noacc
	AEC_gpsrange_shpcirc
	AEC_gpsrange_shppoly
	AEC_gpsrange_shprect
	AEC_gpsrange_shpbad
	AEC_gpsrange_list

	// gps/scan

	AEC_gpsscan_nobind
	AEC_gpsscan_badacc
	AEC_gpsscan_noacc

	// stat/usrlst

	AEC_usrlst_nobind
	AEC_usrlst_asts
	AEC_usrlst_fost
	AEC_usrlst_post
)

// HTTP error messages
var (
	Err404     = errors.New("page not found")
	Err405     = errors.New("method not allowed")
	ErrNotSys  = errors.New("root PUID does not refers to file system path")
	ErrPathOut = errors.New("path cannot refers outside root PUID")

	ErrArgNoHD   = errors.New("'hd' parameter not recognized")
	ErrArgNoDim  = errors.New("bad tiles dimensions")
	ErrArgZDim   = errors.New("dimensions can not be zero")
	ErrNotDir    = errors.New("path is not directory")
	ErrNoPath    = errors.New("path is not found")
	ErrDeny      = errors.New("access denied for specified authorization")
	ErrNotShared = errors.New("access to specified resource does not shared")
	ErrHidden    = errors.New("access to specified file path is disabled")
	ErrNoAccess  = errors.New("profile has no access to specified file path")
	ErrNoCat     = errors.New("specified category does not found")
	ErrNotPlay   = errors.New("file can not be read as playlist")
	ErrFileOver  = errors.New("to many files with same names contains")
	ErrShapeCirc = errors.New("circle must contains 1 coordinates point")
	ErrShapePoly = errors.New("polygon must contains 3 coordinates points at least")
	ErrShapeRect = errors.New("rectangle must contains 4 coordinates points")
	ErrShapeBad  = errors.New("shape is not recognized")
)
