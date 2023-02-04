package hms

import (
	"path"
	"reflect"
	"unsafe"

	"github.com/schwarzlichtbezirk/wpk"
)

// CheckPath is short variant of path existence check.
func CheckPath(fpath string, fname string) (string, bool) {
	if ok, _ := wpk.PathExists(path.Join(fpath, fname)); !ok {
		return "", false
	}
	return fpath, true
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return b
}
