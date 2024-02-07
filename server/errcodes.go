package hms

import "errors"

// Service provider error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	SEC_null = iota
	SEC_noreq
	SEC_badjson
	SEC_badyaml
	SEC_badxml
	SEC_argundef
	SEC_badenc
	SEC_panic

	// authorization
	SEC_noaid
	SEC_noauth

	SEC_auth_absent
	SEC_auth_scheme
	SEC_basic_decode
	SEC_basic_noacc
	SEC_basic_deny
	SEC_token_noacc
	SEC_token_malform
	SEC_token_notsign
	SEC_token_badclaims
	SEC_token_expired
	SEC_token_notyet
	SEC_token_issuer
	SEC_token_error
	SEC_token_badaid
	SEC_param_noacc

	// 404
	SEC_nourl

	// page

	SEC_page_absent

	// file

	SEC_media_badacc
	SEC_media_noacc
	SEC_media_badmedia
	SEC_media_badhd
	SEC_media_badpath
	SEC_media_hidden
	SEC_media_access
	SEC_media_hdgone
	SEC_media_hdfail
	SEC_media_hdnocnt
	SEC_media_medgone
	SEC_media_medfail
	SEC_media_mednocnt
	SEC_media_filegone
	SEC_media_fileopen

	// etmb

	SEC_etmb_badacc
	SEC_etmb_noacc
	SEC_etmb_nopuid
	SEC_etmb_nopath
	SEC_etmb_hidden
	SEC_etmb_access
	SEC_etmb_badcnt
	SEC_etmb_notmb

	// mtmb

	SEC_mtmb_badacc
	SEC_mtmb_noacc
	SEC_mtmb_nopuid
	SEC_mtmb_nopath
	SEC_mtmb_hidden
	SEC_mtmb_access
	SEC_mtmb_badcnt
	SEC_mtmb_absent

	// tile

	SEC_tile_badacc
	SEC_tile_noacc
	SEC_tile_nopuid
	SEC_tile_twodim
	SEC_tile_badwdh
	SEC_tile_badhgt
	SEC_tile_zero
	SEC_tile_nopath
	SEC_tile_hidden
	SEC_tile_access
	SEC_tile_badcnt
	SEC_tile_absent

	// reload

	SEC_reload_load
	SEC_reload_tmpl

	// stat/getlog

	SEC_getlog_nobind

	// auth/pubkey

	SEC_pubkey_rand

	// auth/signin

	SEC_signin_nodata
	SEC_signin_noacc
	SEC_signin_pkey
	SEC_signin_deny

	// auth/refrsh

	SEC_refrsh_nodata
	SEC_refrsh_parse

	// res/folder

	SEC_folder_noacc
	SEC_folder_nodata
	SEC_folder_badpath
	SEC_folder_hidden
	SEC_folder_access
	SEC_folder_stat
	SEC_folder_noshr
	SEC_folder_home
	SEC_folder_drives
	SEC_folder_remote
	SEC_folder_shares
	SEC_folder_media
	SEC_folder_map
	SEC_folder_nocat
	SEC_folder_absent
	SEC_folder_fail
	SEC_folder_open
	SEC_folder_m3u
	SEC_folder_wpl
	SEC_folder_pls
	SEC_folder_asx
	SEC_folder_xspf
	SEC_folder_format
	SEC_folder_tracks

	// res/tags
	SEC_tags_noacc
	SEC_tags_nodata
	SEC_tags_badpath
	SEC_tags_hidden
	SEC_tags_access
	SEC_tags_extract

	// res/ispath

	SEC_ispath_noacc
	SEC_ispath_deny
	SEC_ispath_nodata

	// tags/chk

	SEC_tagschk_nobind
	SEC_tagschk_badacc
	SEC_tagschk_noacc

	// tags/start

	SEC_tagsstart_nobind

	// tags/break

	SEC_tagsbreak_nobind

	// tile/chk

	SEC_tilechk_nobind
	SEC_tilechk_badacc
	SEC_tilechk_noacc

	// tile/start

	SEC_tilestart_nobind

	// tile/break

	SEC_tilebreak_nobind

	// drive/add

	SEC_drvadd_noacc
	SEC_drvadd_deny
	SEC_drvadd_nodata
	SEC_drvadd_badpath
	SEC_drvadd_miss
	SEC_drvadd_hidden

	// drive/del

	SEC_drvdel_noacc
	SEC_drvdel_deny
	SEC_drvdel_nodata
	SEC_drvdel_nopath

	// cloud/add

	SEC_cldadd_noacc
	SEC_cldadd_nodata
	SEC_cldadd_badhost
	SEC_cldadd_ftpdial
	SEC_cldadd_ftpcred
	SEC_cldadd_ftproot
	SEC_cldadd_sftpdial
	SEC_cldadd_sftpcli
	SEC_cldadd_sftppwd
	SEC_cldadd_sftproot
	SEC_cldadd_davdial

	// cloud/del

	SEC_clddel_noacc
	SEC_clddel_deny
	SEC_clddel_nodata
	SEC_clddel_nopath

	// share/add

	SEC_shradd_noacc
	SEC_shradd_deny
	SEC_shradd_nodata
	SEC_shradd_nopath
	SEC_shradd_access

	// share/del

	SEC_shrdel_noacc
	SEC_shrdel_deny
	SEC_shrdel_nodata
	SEC_shrdel_nopath
	SEC_shrdel_access

	// edit/copy

	SEC_edtcopy_nodata
	SEC_edtcopy_noacc
	SEC_edtcopy_deny
	SEC_edtcopy_nopath
	SEC_edtcopy_nodest
	SEC_edtcopy_over
	SEC_edtcopy_opsrc
	SEC_edtcopy_statsrc
	SEC_edtcopy_mkdir
	SEC_edtcopy_rd
	SEC_edtcopy_opdst
	SEC_edtcopy_copy
	SEC_edtcopy_statfile

	// edit/rename

	SEC_edtren_nodata
	SEC_edtren_noacc
	SEC_edtren_deny
	SEC_edtren_nopath
	SEC_edtren_nodest
	SEC_edtren_over
	SEC_edtren_move
	SEC_edtren_stat

	// edit/del

	SEC_edtdel_nodata
	SEC_edtdel_noacc
	SEC_edtdel_deny
	SEC_edtdel_nopath
	SEC_edtdel_remove

	// gps/range
	SEC_gpsrange_noacc
	SEC_gpsrange_shpcirc
	SEC_gpsrange_shppoly
	SEC_gpsrange_shprect
	SEC_gpsrange_shpbad
	SEC_gpsrange_list

	// gps/scan
	SEC_gpsscan_noacc
	SEC_gpsscan_nodata

	// stat/usrlst
	SEC_usrlst_nobind
	SEC_usrlst_asts
	SEC_usrlst_fost
	SEC_usrlst_post
)

// HTTP error messages
var (
	Err404      = errors.New("page not found")
	ErrNoJSON   = errors.New("data not given")
	ErrNoData   = errors.New("data is empty")
	ErrArgUndef = errors.New("request content type is undefined")
	ErrBadEnc   = errors.New("encoding format does not supported")
	ErrNotSys   = errors.New("root PUID does not refers to file system path")
	ErrPathOut  = errors.New("path cannot refers outside root PUID")

	ErrArgNoHD   = errors.New("'hd' parameter not recognized")
	ErrArgNoPuid = errors.New("'puid' argument required")
	ErrArgNoDim  = errors.New("bad tiles dimensions")
	ErrArgZDim   = errors.New("dimensions can noy be zero")
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
