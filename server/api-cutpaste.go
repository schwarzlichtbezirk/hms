package hms

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
)

// APIHANDLER
func edtcopyAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src"`
		Dst string `json:"dst" yaml:"dst" xml:"dst"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		FT FT_t `json:"ft" yaml:"ft" xml:"ft"`
	}
	var isret bool
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" || arg.Dst == "" {
		WriteError400(w, r, ErrArgNoPuid, SEC_edtcopy_nodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = ProfileByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, SEC_edtcopy_noacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, SEC_edtcopy_deny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, SEC_edtcopy_nopath)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		WriteError(w, r, http.StatusNotFound, err, SEC_edtcopy_nodest)
		return
	}
	dstpath = JoinPath(dstpath, path.Base(srcpath))

	// copies file or dir from source to destination
	var filecopy func(srcpath, dstpath string) (err error)
	filecopy = func(srcpath, dstpath string) (err error) {
		// generate unique destination filename
		if !arg.Ovw {
			var ext = path.Ext(dstpath)
			var org = dstpath[:len(dstpath)-len(ext)]
			var i = 1
			for {
				if _, err = os.Stat(dstpath); errors.Is(err, fs.ErrNotExist) {
					break
				}
				i++
				dstpath = fmt.Sprintf("%s (%d)%s", org, i, ext)
				if i > 100 {
					err = ErrFileOver
					WriteError500(w, r, err, SEC_edtcopy_over)
					return
				}
			}
		}

		var src fs.File
		var dst *os.File
		var fi fs.FileInfo
		// open source file
		if src, err = JP.Open(srcpath); err != nil {
			WriteError500(w, r, err, SEC_edtcopy_opsrc)
			return
		}
		defer func() {
			src.Close()
			if fi != nil {
				os.Chtimes(dstpath, fi.ModTime(), fi.ModTime())
			}
		}()

		if fi, err = src.Stat(); err != nil {
			WriteError500(w, r, err, SEC_edtcopy_statsrc)
			return
		}
		if fi.IsDir() {
			// create destination dir
			if err = os.Mkdir(dstpath, 0644); err != nil && !errors.Is(err, fs.ErrExist) {
				WriteError500(w, r, err, SEC_edtcopy_mkdir)
				return
			}

			// get returned dir properties now
			if !isret {
				ret.FT = FTdir
				isret = true
			}

			// copy dir content
			var files []fs.DirEntry
			if files, err = JP.ReadDir(srcpath); err != nil {
				WriteError500(w, r, err, SEC_edtcopy_rd)
				return
			}
			for _, file := range files {
				var name = file.Name()
				if err = filecopy(JoinPath(srcpath, name), JoinPath(dstpath, name)); err != nil {
					return // error already written
				}
			}
		} else {
			// create destination file
			if dst, err = os.Create(dstpath); err != nil {
				WriteError500(w, r, err, SEC_edtcopy_opdst)
				return
			}
			defer dst.Close()

			// copy file content
			if _, err = io.Copy(dst, src); err != nil {
				WriteError500(w, r, err, SEC_edtcopy_copy)
				return
			}

			// get returned file properties at last
			if !isret {
				isret = true
				ret.FT = FTfile
			}
		}
		return
	}
	if err = filecopy(srcpath, dstpath); err != nil {
		return // error already written
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func edtrenameAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src"`
		Dst string `json:"dst" yaml:"dst" xml:"dst"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Prop FileKit `json:"prop" yaml:"prop" xml:"prop"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" || arg.Dst == "" {
		WriteError400(w, r, ErrArgNoPuid, SEC_edtren_nodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = ProfileByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, SEC_edtren_noacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, SEC_edtren_deny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, SEC_edtren_nopath)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		WriteError(w, r, http.StatusNotFound, err, SEC_edtren_nodest)
		return
	}
	dstpath = JoinPath(dstpath, path.Base(srcpath))

	// generate unique destination filename
	if !arg.Ovw {
		var ext = path.Ext(dstpath)
		var org = dstpath[:len(dstpath)-len(ext)]
		var i = 1
		for {
			if _, err = os.Stat(dstpath); errors.Is(err, fs.ErrNotExist) {
				break
			}
			i++
			dstpath = fmt.Sprintf("%s (%d)%s", org, i, ext)
			if i > 100 {
				err = ErrFileOver
				WriteError500(w, r, err, SEC_edtren_over)
				return
			}
		}
	}

	// rename destination file
	if err = os.Rename(srcpath, dstpath); err != nil && !errors.Is(err, fs.ErrExist) {
		WriteError500(w, r, err, SEC_edtren_move)
		return
	}

	// get returned file properties at last
	var fi fs.FileInfo
	if fi, err = JP.Stat(dstpath); err != nil {
		WriteError500(w, r, err, SEC_edtren_stat)
		return
	}
	ret.Prop.PuidProp.Setup(session, dstpath)
	ret.Prop.FileProp.Setup(fi)
	if tp, ok := tilecache.Peek(ret.Prop.PUID); ok {
		ret.Prop.TileProp = tp
	}
	if xp, ok := extcache.Peek(ret.Prop.PUID); ok {
		ret.Prop.ExtProp = xp
	}

	WriteOK(w, r, &ret)
}

// APIHANDLER
func edtdeleteAPI(w http.ResponseWriter, r *http.Request, aid, uid ID_t) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src"`
	}
	if uid == 0 { // only authorized access allowed
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, SEC_noauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" {
		WriteError400(w, r, ErrArgNoPuid, SEC_edtdel_nodata)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = ProfileByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, SEC_edtdel_noacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, SEC_edtdel_deny)
		return
	}

	var srcpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, SEC_edtdel_nopath)
		return
	}

	if err = os.RemoveAll(srcpath); err != nil {
		WriteError500(w, r, err, SEC_edtdel_remove)
		return
	}

	WriteOK(w, r, nil)
}

// The End.
