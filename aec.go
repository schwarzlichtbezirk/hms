package hms

import "errors"

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AECnull = iota
	AECnoreq
	AECbadjson
	AECbadyaml
	AECbadxml
	AECargundef
	AECbadenc
	AECpanic

	// auth

	AECnoaid
	AECnoauth
	AECtokenmalform
	AECtokennotsign
	AECtokenexpired
	AECtokennotyet
	AECtokenerror
	AECtokenless
	AECtokennoacc
	AECtokennoaid

	// page

	AECpageabsent
	AECfileabsent

	// file

	AECmedianoacc
	AECmediabadmedia
	AECmediabadhd
	AECmediabadpath
	AECmediahidden
	AECmediaaccess
	AECmediahdgone
	AECmediahdfail
	AECmediahdnocnt
	AECmediamedgone
	AECmediamedfail
	AECmediamednocnt
	AECmediafilegone
	AECmediafileopen

	// etmb

	AECetmbnoacc
	AECetmbnopuid
	AECetmbnopath
	AECetmbhidden
	AECetmbaccess
	AECetmbbadcnt
	AECetmbnotmb

	// mtmb

	AECmtmbnoacc
	AECmtmbnopuid
	AECmtmbnopath
	AECmtmbhidden
	AECmtmbaccess
	AECmtmbbadcnt
	AECmtmbnocnt

	// tile

	AECtilenoacc
	AECtilenopuid
	AECtilebadres
	AECtilenopath
	AECtilehidden
	AECtileaccess
	AECtilebadcnt
	AECtilenocnt

	// reload

	AECreloadload
	AECreloadtmpl

	// stat/getlog

	AECgetlogbadnum
	AECgetlogbadunix
	AECgetlogbadums

	// auth/pubkey

	AECpubkeyrand

	// auth/signin

	AECsigninnodata
	AECsigninnoacc
	AECsigninpkey
	AECsignindeny

	// auth/refrsh

	AECrefrshnodata
	AECrefrshparse

	// res/folder

	AECfoldernoacc
	AECfoldernodata
	AECfolderbadpath
	AECfolderhidden
	AECfolderaccess
	AECfolderstat
	AECfoldernoshr
	AECfolderhome
	AECfolderdrives
	AECfolderremote
	AECfoldershares
	AECfoldermedia
	AECfoldermap
	AECfoldernocat
	AECfolderabsent
	AECfolderfail
	AECfolderopen
	AECfolderm3u
	AECfolderwpl
	AECfolderpls
	AECfolderasx
	AECfolderxspf
	AECfolderformat
	AECfoldertracks

	// res/tags
	AECtagsnoacc
	AECtagsnodata
	AECtagsbadpath
	AECtagshidden
	AECtagsaccess
	AECtagsnoexif
	AECtagsnoid3
	AECtagsnotags

	// res/ispath

	AECispathnoacc
	AECispathdeny
	AECispathnodata

	// tile/chk

	AECtilechknoacc
	AECtilechknodata

	// tile/scnstart

	AECscnstartnoacc
	AECscnstartnodata

	// tile/scnbreak

	AECscnbreaknoacc
	AECscnbreaknoaid
	AECscnbreaknodata

	// drive/add

	AECdrvaddnoacc
	AECdrvadddeny
	AECdrvaddnodata
	AECdrvaddbadpath
	AECdrvaddmiss
	AECdrvaddhidden

	// drive/del

	AECdrvdelnoacc
	AECdrvdeldeny
	AECdrvdelnodata
	AECdrvdelnopath

	// cloud/add

	AECcldaddnoacc
	AECcldaddnodata
	AECcldaddftpdial
	AECcldaddftpcred
	AECcldaddftproot
	AECcldaddsftpdial
	AECcldaddsftpcli
	AECcldaddsftppwd
	AECcldaddsftproot
	AECcldadddavdial

	// cloud/del

	AECclddelnoacc
	AECclddeldeny
	AECclddelnodata
	AECclddelnopath

	// share/add

	AECshraddnoacc
	AECshradddeny
	AECshraddnodata
	AECshraddnopath
	AECshraddaccess

	// share/del

	AECshrdelnoacc
	AECshrdeldeny
	AECshrdelnodata
	AECshrdelnopath
	AECshrdelaccess

	// edit/copy

	AECedtcopynodata
	AECedtcopynoacc
	AECedtcopydeny
	AECedtcopynopath
	AECedtcopynodest
	AECedtcopyover
	AECedtcopyopsrc
	AECedtcopystatsrc
	AECedtcopymkdir
	AECedtcopyrd
	AECedtcopyopdst
	AECedtcopycopy
	AECedtcopystatfile

	// edit/rename

	AECedtrennodata
	AECedtrennoacc
	AECedtrendeny
	AECedtrennopath
	AECedtrennodest
	AECedtrenover
	AECedtrenmove
	AECedtrenstat

	// edit/del

	AECedtdelnodata
	AECedtdelnoacc
	AECedtdeldeny
	AECedtdelnopath
	AECedtdelremove

	// gps/range
	AECgpsrangenoacc
	AECgpsrangeshpcirc
	AECgpsrangeshppoly
	AECgpsrangeshprect
	AECgpsrangeshpbad
	AECgpsrangelist

	// gps/scan
	AECgpsscannoacc
	AECgpsscannodata

	// stat/usrlst
	AECusrlstasts
	AECusrlstfost
	AECusrlstpost
)

// HTTP error messages
var (
	ErrNoJSON   = errors.New("data not given")
	ErrNoData   = errors.New("data is empty")
	ErrArgUndef = errors.New("request content type is undefined")
	ErrBadEnc   = errors.New("encoding format does not supported")
	ErrNotSys   = errors.New("root PUID does not refers to file system path")
	ErrPathOut  = errors.New("path cannot refers outside root PUID")

	ErrNotFound  = errors.New("resource not found")
	ErrArgNoNum  = errors.New("'num' parameter not recognized")
	ErrArgNoTime = errors.New("unix time value not recognized")
	ErrArgNoHD   = errors.New("'hd' parameter not recognized")
	ErrArgNoPuid = errors.New("'puid' argument required")
	ErrArgNoRes  = errors.New("bad tiles resolution")
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
