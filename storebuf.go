package hms

import (
	"errors"
)

var (
	ErrBadType = errors.New("type does not supported to insert into database")
)

type StoreBuf struct {
	extbuf  []ExtStore
	exifbuf []ExifStore
	id3buf  []Id3Store
}

func (buf *StoreBuf) Init() {
	const limit = 256
	buf.extbuf = make([]ExtStore, 0, limit)
	buf.exifbuf = make([]ExifStore, 0, limit)
	buf.id3buf = make([]Id3Store, 0, limit)
}

func (buf *StoreBuf) Push(session *Session, val any) (err error) {
	if buf == nil {
		return
	}
	switch st := val.(type) {
	case ExtStore:
		buf.extbuf = append(buf.extbuf, st)
		if len(buf.extbuf) == cap(buf.extbuf) {
			if _, err = session.Insert(&buf.extbuf); err != nil {
				return
			}
			buf.extbuf = buf.extbuf[:0]
		}
	case ExifStore:
		buf.exifbuf = append(buf.exifbuf, st)
		if len(buf.exifbuf) == cap(buf.exifbuf) {
			if _, err = session.Insert(&buf.exifbuf); err != nil {
				return
			}
			buf.exifbuf = buf.exifbuf[:0]
		}
	case Id3Store:
		buf.id3buf = append(buf.id3buf, st)
		if len(buf.id3buf) == cap(buf.id3buf) {
			if _, err = session.Insert(&buf.id3buf); err != nil {
				return
			}
			buf.id3buf = buf.id3buf[:0]
		}
	default:
		return ErrBadType
	}
	return
}

func (buf *StoreBuf) Flush(session *Session) (err error) {
	if buf == nil {
		return
	}
	if len(buf.extbuf) > 0 {
		if _, err = session.Insert(&buf.extbuf); err != nil {
			return
		}
		buf.extbuf = buf.extbuf[:0]
	}
	if len(buf.exifbuf) > 0 {
		if _, err = session.Insert(&buf.exifbuf); err != nil {
			return
		}
		buf.exifbuf = buf.exifbuf[:0]
	}
	if len(buf.id3buf) > 0 {
		if _, err = session.Insert(&buf.id3buf); err != nil {
			return
		}
		buf.id3buf = buf.id3buf[:0]
	}
	return
}

// The End.
