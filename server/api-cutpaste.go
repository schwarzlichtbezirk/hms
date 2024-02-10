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

	"github.com/gin-gonic/gin"
)

// APIHANDLER
func SpiEditCopy(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src" binding:"required"`
		Dst string `json:"dst" yaml:"dst" xml:"dst" binding:"required"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		FT FT_t `json:"ft" yaml:"ft" xml:"ft"`
	}
	var isret bool

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_edtcopy_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_edtcopy_badacc, ErrNoAcc)
		return
	}
	if !Profiles.Has(aid) {
		Ret404(c, SEC_edtcopy_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, SEC_edtcopy_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		Ret404(c, SEC_edtcopy_nopath, err)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		Ret404(c, SEC_edtcopy_nodest, err)
		return
	}
	dstpath = JoinPath(dstpath, path.Base(srcpath))

	// copies file or dir from source to destination
	var filecopy func(srcpath, dstpath string) (code int, err error)
	filecopy = func(srcpath, dstpath string) (code int, err error) {
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
					return SEC_edtcopy_over, ErrFileOver
				}
			}
		}

		var src fs.File
		var dst *os.File
		var fi fs.FileInfo
		// open source file
		if src, err = JP.Open(srcpath); err != nil {
			return SEC_edtcopy_opsrc, err
		}
		defer func() {
			src.Close()
			if fi != nil {
				os.Chtimes(dstpath, fi.ModTime(), fi.ModTime())
			}
		}()

		if fi, err = src.Stat(); err != nil {
			return SEC_edtcopy_statsrc, err
		}
		if fi.IsDir() {
			// create destination dir
			if err = os.Mkdir(dstpath, 0644); err != nil && !errors.Is(err, fs.ErrExist) {
				return SEC_edtcopy_mkdir, err
			}

			// get returned dir properties now
			if !isret {
				ret.FT = FTdir
				isret = true
			}

			// copy dir content
			var files []fs.DirEntry
			if files, err = JP.ReadDir(srcpath); err != nil {
				return SEC_edtcopy_rd, err
			}
			for _, file := range files {
				var name = file.Name()
				if code, err = filecopy(JoinPath(srcpath, name), JoinPath(dstpath, name)); err != nil {
					return // error already written
				}
			}
		} else {
			// create destination file
			if dst, err = os.Create(dstpath); err != nil {
				return SEC_edtcopy_opdst, err
			}
			defer dst.Close()

			// copy file content
			if _, err = io.Copy(dst, src); err != nil {
				return SEC_edtcopy_copy, err
			}

			// get returned file properties at last
			if !isret {
				isret = true
				ret.FT = FTfile
			}
		}
		return
	}
	if code, err := filecopy(srcpath, dstpath); err != nil {
		Ret500(c, code, err)
		return
	}

	RetOk(c, ret)
}

// APIHANDLER
func SpiEditRename(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src" binding:"required"`
		Dst string `json:"dst" yaml:"dst" xml:"dst" binding:"required"`
		Ovw bool   `json:"overwrite,omitempty" yaml:"overwrite,omitempty" xml:"overwrite,omitempty,attr"`
	}
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Prop FileKit `json:"prop" yaml:"prop" xml:"prop"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_edtren_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_edtren_badacc, ErrNoAcc)
		return
	}
	if !Profiles.Has(aid) {
		Ret404(c, SEC_edtren_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, SEC_edtren_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	if !Profiles.Has(aid) {
		Ret400(c, SEC_edtren_noacc, ErrNoAcc)
		return
	}
	if uid != aid {
		Ret403(c, SEC_edtren_deny, ErrDeny)
		return
	}

	// get source and destination filenames
	var srcpath, dstpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		Ret404(c, SEC_edtren_nopath, err)
		return
	}
	if dstpath, _, err = UnfoldPath(session, arg.Dst); err != nil {
		Ret404(c, SEC_edtren_nodest, err)
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
				Ret500(c, SEC_edtren_over, err)
				return
			}
		}
	}

	// rename destination file
	if err = os.Rename(srcpath, dstpath); err != nil && !errors.Is(err, fs.ErrExist) {
		Ret500(c, SEC_edtren_move, err)
		return
	}

	// get returned file properties at last
	var fi fs.FileInfo
	if fi, err = JP.Stat(dstpath); err != nil {
		Ret500(c, SEC_edtren_stat, err)
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

	RetOk(c, ret)
}

// APIHANDLER
func SpiEditDelete(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Src string `json:"src" yaml:"src" xml:"src" binding:"required"`
	}

	// get arguments
	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_edtdel_nobind, err)
		return
	}
	var uid = GetUID(c)
	var aid uint64
	if aid, err = ParseID(c.Param("aid")); err != nil {
		Ret400(c, SEC_edtdel_badacc, ErrNoAcc)
		return
	}
	if !Profiles.Has(aid) {
		Ret404(c, SEC_edtdel_noacc, ErrNoAcc)
		return
	}

	if uid != aid {
		Ret403(c, SEC_edtdel_deny, ErrDeny)
		return
	}

	var session = XormStorage.NewSession()
	defer session.Close()

	if !Profiles.Has(aid) {
		Ret400(c, SEC_edtdel_noacc, ErrNoAcc)
		return
	}
	if uid != aid {
		Ret403(c, SEC_edtdel_deny, ErrDeny)
		return
	}

	var srcpath string
	if srcpath, _, err = UnfoldPath(session, arg.Src); err != nil {
		Ret404(c, SEC_edtdel_nopath, err)
		return
	}

	if err = os.RemoveAll(srcpath); err != nil {
		Ret500(c, SEC_edtdel_remove, err)
		return
	}

	c.Status(http.StatusOK)
}

// The End.
