package hms

import "errors"

// API error codes.
// Each error code have unique source code point,
// so this error code at service reply exactly points to error place.
const (
	AECnull = iota
	AECbadbody
	AECnoreq
	AECbadjson
	AECpanic

	// auth

	AECnoauth
	AECtokenless
	AECtokenerror
	AECtokennoacc

	// page

	AECpageabsent
	AECfileabsent

	// file

	AECmediabadmedia
	AECmediabadhd
	AECmediabadaccid
	AECmedianoacc
	AECmediaroot
	AECmedianopath
	AECmediahidden
	AECmedianoprop
	AECmedianofile
	AECmediaaccess
	AECmediahdgone
	AECmediahdfail
	AECmediahdnocnt
	AECmediamedgone
	AECmediamedfail
	AECmediamednocnt
	AECmediafilegone
	AECmediafileopen

	// thumb

	AECthumbnoacc
	AECthumbnopuid
	AECthumbnopath
	AECthumbhidden
	AECthumbnoprop
	AECthumbnofile
	AECthumbaccess
	AECthumbabsent
	AECthumbbadcnt

	// tile

	AECtilenoacc
	AECtilenopuid
	AECtilenopath
	AECtilehidden
	AECtilenoprop
	AECtilenofile
	AECtileaccess
	AECtileabsent
	AECtilebadcnt

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

	// res/ishome

	AECishomenoacc

	// res/ctgr

	AECctgrnodata
	AECctgrnopath
	AECctgrnoacc
	AECctgrnoshr
	AECctgrnotcat

	// res/folder

	AECfoldernodata
	AECfoldernoacc
	AECfolderroot
	AECfoldernopath
	AECfolderhidden
	AECfolderaccess
	AECfolderstat
	AECfolderopen
	AECfolderm3u
	AECfolderwpl
	AECfolderpls
	AECfolderasx
	AECfolderxspf
	AECfolderformat
	AECfolderabsent
	AECfolderfail

	// res/ispath

	AECispathnoacc
	AECispathdeny
	AECispathroot
	AECispathhidden

	// tmb/chk

	AECtmbchknodata

	// tmb/scnstart

	AECscnstartnodata
	AECscnstartnoacc

	// tmb/scnbreak

	AECscnbreaknodata
	AECscnbreaknoacc

	// share/add

	AECshraddnodata
	AECshraddnoacc
	AECshradddeny
	AECshraddnopath
	AECshraddaccess

	// share/del

	AECshrdelnodata
	AECshrdelnoacc
	AECshrdeldeny

	// drive/add

	AECdrvaddnodata
	AECdrvaddnoacc
	AECdrvadddeny
	AECdrvaddroot
	AECdrvaddfile

	// drive/del

	AECdrvdelnodata
	AECdrvdelnoacc
	AECdrvdeldeny
	AECdrvdelnopath

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
)

// HTTP error messages
var (
	ErrNoJSON = errors.New("data not given")
	ErrNoData = errors.New("data is empty")

	ErrNotFound  = errors.New("404 page not found")
	ErrArgNoNum  = errors.New("'num' parameter not recognized")
	ErrArgNoHD   = errors.New("'hd' parameter not recognized")
	ErrArgNoCid  = errors.New("'cid' parameter not recognized")
	ErrArgNoPuid = errors.New("'puid' argument required")
	ErrNotDir    = errors.New("path is not directory")
	ErrNoPath    = errors.New("path is not found")
	ErrDeny      = errors.New("access denied for specified authorization")
	ErrNotShared = errors.New("access to specified resource does not shared")
	ErrHidden    = errors.New("access to specified file path is disabled")
	ErrNoAccess  = errors.New("profile has no access to specified file path")
	ErrNotCat    = errors.New("only categories can be accepted")
	ErrNotPlay   = errors.New("file can not be read as playlist")
	ErrFileOver  = errors.New("to many files with same names contains")
)
