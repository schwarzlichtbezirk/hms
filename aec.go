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

	AECmedianoaid
	AECmediabadmedia
	AECmediabadhd
	AECmediabadpath
	AECmedianoacc
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

	AECetmbnoaid
	AECetmbnopuid
	AECetmbnoacc
	AECetmbnopath
	AECetmbhidden
	AECetmbaccess
	AECetmbbadcnt
	AECetmbnotmb

	// mtmb

	AECmtmbnoaid
	AECmtmbnopuid
	AECmtmbnoacc
	AECmtmbnopath
	AECmtmbhidden
	AECmtmbaccess
	AECmtmbbadcnt
	AECmtmbnocnt

	// tile

	AECtilenoaid
	AECtilenopuid
	AECtilebadres
	AECtilenoacc
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

	AECfoldernoaid
	AECfoldernodata
	AECfoldernoacc
	AECfolderbadpath
	AECfolderhidden
	AECfolderaccess
	AECfolderstat
	AECfoldernoshr
	AECfolderhome
	AECfolderdrives
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
	AECtagsnoaid
	AECtagsnodata
	AECtagsnoacc
	AECtagsbadpath
	AECtagshidden
	AECtagsaccess
	AECtagsnoexif
	AECtagsnoid3
	AECtagsnotags

	// res/ispath

	AECispathnoaid
	AECispathnodata
	AECispathnoacc
	AECispathdeny
	AECispathbadpath
	AECispathmiss
	AECispathhidden

	// tile/chk

	AECtilechknoaid
	AECtilechknodata

	// tile/scnstart

	AECscnstartnoaid
	AECscnstartnodata
	AECscnstartnoacc

	// tile/scnbreak

	AECscnbreaknoaid
	AECscnbreaknodata
	AECscnbreaknoacc

	// share/add

	AECshraddnoaid
	AECshraddnodata
	AECshraddnoacc
	AECshradddeny
	AECshraddnopath
	AECshraddaccess

	// share/del

	AECshrdelnoaid
	AECshrdelnodata
	AECshrdelnoacc
	AECshrdeldeny

	// drive/add

	AECdrvaddnoaid
	AECdrvaddnodata
	AECdrvaddnoacc
	AECdrvadddeny
	AECdrvaddbadpath
	AECdrvaddmiss
	AECdrvaddhidden

	// drive/del

	AECdrvdelnoaid
	AECdrvdelnodata
	AECdrvdelnoacc
	AECdrvdeldeny
	AECdrvdelnopath

	// edit/copy

	AECedtcopynoaid
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

	AECedtrennoaid
	AECedtrennodata
	AECedtrennoacc
	AECedtrendeny
	AECedtrennopath
	AECedtrennodest
	AECedtrenover
	AECedtrenmove
	AECedtrenstat

	// edit/del

	AECedtdelnoaid
	AECedtdelnodata
	AECedtdelnoacc
	AECedtdeldeny
	AECedtdelnopath
	AECedtdelremove

	// gps/range
	AECgpsrangenoaid
	AECgpsrangeshpcirc
	AECgpsrangeshppoly
	AECgpsrangeshprect
	AECgpsrangeshpbad
	AECgpsrangenoacc
	AECgpsrangelist

	// gps/scan
	AECgpsscannoaid
	AECgpsscannodata
	AECgpsscannoacc

	// stat/usrlst
	AECusrlstusts
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
