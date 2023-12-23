package hms

import (
	"errors"
)

var (
	ErrBadType = errors.New("type does not supported to insert into database")
)

func UpsertBuffer[T any](session *Session, table any, buf *[]Store[T]) (err error) {
	if session != nil {
		if _, err = session.Table(table).Insert(buf); err != nil {
			for _, val := range *buf {
				if _, err = session.Table(table).ID(val.Puid).Update(&val); err != nil {
					return
				}
			}
		}
	}
	*buf = (*buf)[:0]
	return
}

type StoreBuf struct {
	extbuf  []Store[ExtProp]
	exifbuf []Store[ExifProp]
	id3buf  []Store[Id3Prop]
}

func (sb *StoreBuf) Init(limit int) {
	sb.extbuf = make([]Store[ExtProp], 0, limit)
	sb.exifbuf = make([]Store[ExifProp], 0, limit)
	sb.id3buf = make([]Store[Id3Prop], 0, limit)
}

func (sb *StoreBuf) Push(session *Session, val any) (err error) {
	if sb == nil {
		return
	}
	switch st := val.(type) {
	case ExtStore:
		sb.extbuf = append(sb.extbuf, Store[ExtProp](st))
		if len(sb.extbuf) == cap(sb.extbuf) {
			err = UpsertBuffer(session, ExtStore{}, &sb.extbuf)
		}
	case ExifStore:
		sb.exifbuf = append(sb.exifbuf, Store[ExifProp](st))
		if len(sb.exifbuf) == cap(sb.exifbuf) {
			err = UpsertBuffer(session, ExifStore{}, &sb.exifbuf)
		}
	case Id3Store:
		sb.id3buf = append(sb.id3buf, Store[Id3Prop](st))
		if len(sb.id3buf) == cap(sb.id3buf) {
			err = UpsertBuffer(session, Id3Store{}, &sb.id3buf)
		}
	default:
		return ErrBadType
	}
	return
}

func (sb *StoreBuf) Flush(session *Session) (err error) {
	if sb == nil {
		return
	}
	var errs [3]error
	if len(sb.extbuf) > 0 {
		errs[0] = UpsertBuffer(session, ExtStore{}, &sb.extbuf)
	}
	if len(sb.exifbuf) > 0 {
		errs[1] = UpsertBuffer(session, ExifStore{}, &sb.exifbuf)
	}
	if len(sb.id3buf) > 0 {
		errs[2] = UpsertBuffer(session, Id3Store{}, &sb.id3buf)
	}
	return errors.Join(errs[:]...)
}

// The End.
