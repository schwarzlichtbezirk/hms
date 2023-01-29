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
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" || arg.Dst == "" {
		WriteError400(w, r, ErrArgNoPuid, AECedtcopynodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtcopynoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtcopydeny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECedtcopynopath)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECedtcopynodest)
		return
	}
	dstpath = path.Join(dstpath, path.Base(srcpath))

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
					WriteError500(w, r, err, AECedtcopyover)
					return
				}
			}
		}

		var src, dst *os.File
		var fi fs.FileInfo
		// open source file
		if src, err = os.Open(srcpath); err != nil {
			WriteError500(w, r, err, AECedtcopyopsrc)
			return
		}
		defer func() {
			src.Close()
			if fi != nil {
				os.Chtimes(dstpath, fi.ModTime(), fi.ModTime())
			}
		}()

		if fi, err = src.Stat(); err != nil {
			WriteError500(w, r, err, AECedtcopystatsrc)
			return
		}
		if fi.IsDir() {
			// create destination dir
			if err = os.Mkdir(dstpath, 0644); err != nil && !errors.Is(err, fs.ErrExist) {
				WriteError500(w, r, err, AECedtcopymkdir)
				return
			}

			// get returned dir properties now
			if !isret {
				ret.FT = FTdir
				isret = true
			}

			// copy dir content
			var files []fs.DirEntry
			if files, err = src.ReadDir(-1); err != nil {
				WriteError500(w, r, err, AECedtcopyrd)
				return
			}
			for _, file := range files {
				var name = file.Name()
				if err = filecopy(path.Join(srcpath, name), path.Join(dstpath, name)); err != nil {
					return // error already written
				}
			}
		} else {
			// create destination file
			if dst, err = os.Create(dstpath); err != nil {
				WriteError500(w, r, err, AECedtcopyopdst)
				return
			}
			defer dst.Close()

			// copy file content
			if _, err = io.Copy(dst, src); err != nil {
				WriteError500(w, r, err, AECedtcopycopy)
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
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" || arg.Dst == "" {
		WriteError400(w, r, ErrArgNoPuid, AECedtrennodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtrennoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtrendeny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECedtrennopath)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECedtrennodest)
		return
	}
	dstpath = path.Join(dstpath, path.Base(srcpath))

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
				WriteError500(w, r, err, AECedtrenover)
				return
			}
		}
	}

	// rename destination file
	if err = os.Rename(srcpath, dstpath); err != nil && !errors.Is(err, fs.ErrExist) {
		WriteError500(w, r, err, AECedtrenmove)
		return
	}

	// get returned file properties at last
	var fi fs.FileInfo
	if fi, err = StatFile(dstpath); err != nil {
		WriteError500(w, r, err, AECedtrenstat)
		return
	}
	ret.Prop.Setup(session, dstpath, fi)

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
		WriteError(w, r, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
		return
	}

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Src == "" {
		WriteError400(w, r, ErrArgNoPuid, AECedtdelnodata)
		return
	}

	var session = xormStorage.NewSession()
	defer session.Close()

	var prf *Profile
	if prf = prflist.ByID(aid); prf == nil {
		WriteError400(w, r, ErrNoAcc, AECedtdelnoacc)
		return
	}
	if uid != aid {
		WriteError(w, r, http.StatusForbidden, ErrDeny, AECedtdeldeny)
		return
	}

	var srcpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		WriteError(w, r, http.StatusNotFound, err, AECedtdelnopath)
		return
	}

	if err = os.RemoveAll(srcpath); err != nil {
		WriteError500(w, r, err, AECedtdelremove)
		return
	}

	WriteOK(w, r, nil)
}

// The End.
