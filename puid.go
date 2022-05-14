package hms

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"time"

	"gopkg.in/yaml.v3"
)

// ID_t is the type of any users identifiers
type ID_t uint64

// Puid_t represents integer form of path unique ID.
type Puid_t uint64

// Predefined PUIDs.
const (
	PUIDhome   Puid_t = 1
	PUIDdrives Puid_t = 2
	PUIDshares Puid_t = 3
	PUIDmedia  Puid_t = 4
	PUIDvideo  Puid_t = 5
	PUIDaudio  Puid_t = 6
	PUIDimage  Puid_t = 7
	PUIDbooks  Puid_t = 8
	PUIDtexts  Puid_t = 9

	PUIDreserved = 32
)

// Categories paths constants.
const (
	CPhome   = "<home>"
	CPdrives = "<drives>"
	CPshares = "<shares>"
	CPmedia  = "<media>"
	CPvideo  = "<video>"
	CPaudio  = "<audio>"
	CPimage  = "<image>"
	CPbooks  = "<books>"
	CPtexts  = "<texts>"
)

var CatNames = map[Puid_t]string{
	PUIDhome:   "Home",
	PUIDdrives: "Drives list",
	PUIDshares: "Shared resources",
	PUIDmedia:  "Multimedia files",
	PUIDvideo:  "Movie and video files",
	PUIDaudio:  "Music and audio files",
	PUIDimage:  "Photos and images",
	PUIDbooks:  "Books",
	PUIDtexts:  "Text files",
}

// CatKeyPath is predefined read-only maps with PUIDs keys and categories values.
var CatKeyPath = map[Puid_t]string{
	PUIDhome:   CPhome,
	PUIDdrives: CPdrives,
	PUIDshares: CPshares,
	PUIDmedia:  CPmedia,
	PUIDvideo:  CPvideo,
	PUIDaudio:  CPaudio,
	PUIDimage:  CPimage,
	PUIDbooks:  CPbooks,
	PUIDtexts:  CPtexts,
}

// CatPathKey is predefined read-only map with categories keys and PUIDs values.
var CatPathKey = map[string]Puid_t{
	CPhome:   PUIDhome,
	CPdrives: PUIDdrives,
	CPshares: PUIDshares,
	CPmedia:  PUIDmedia,
	CPvideo:  PUIDvideo,
	CPaudio:  PUIDaudio,
	CPimage:  PUIDimage,
	CPbooks:  PUIDbooks,
	CPtexts:  PUIDtexts,
}

// Produce base32 string representation of given random bytes slice.
var idenc = base32.HexEncoding.WithPadding(base32.NoPadding)

// String converts path unique ID to base32 string representation.
func (pt Puid_t) String() string {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(pt))
	var n int
	for n = 7; n >= 0 && buf[n] == 0; n-- {
	}
	return idenc.EncodeToString(buf[:n+1])
}

// Set writes base32 string representation of ID into integer value.
func (pt *Puid_t) Set(puid string) error {
	var buf [8]byte
	_, err := idenc.Decode(buf[:], []byte(puid))
	*pt = Puid_t(binary.LittleEndian.Uint64(buf[:]))
	return err
}

// MarshalJSON is JSON marshaler interface implementation.
func (pt Puid_t) MarshalJSON() ([]byte, error) {
	return json.Marshal(pt.String())
}

// UnmarshalJSON is JSON unmarshaler interface implementation.
func (pt *Puid_t) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return pt.Set(s)
}

// MarshalYAML is YAML marshaler interface implementation.
func (pt Puid_t) MarshalYAML() (interface{}, error) {
	return pt.String(), nil
}

// UnmarshalYAML is YAML unmarshaler interface implementation.
func (pt *Puid_t) UnmarshalYAML(value *yaml.Node) error {
	return pt.Set(value.Value)
}

// Rand generates random identifier of given length in bits, maximum 64 bits.
// Multiply x5 to convert length in base32 symbols to bits.
func (pt *Puid_t) Rand(bits int) {
	var buf [8]byte
	if _, err := rand.Read(buf[:(bits+7)/8]); err != nil {
		panic(err)
	}
	*pt = Puid_t(binary.LittleEndian.Uint64(buf[:]))
	*pt &= 0xffffffffffffffff >> (65 - bits) // throw one more bit to prevent string representation overflow
}

type unix_t uint64

func (ut unix_t) Time() time.Time {
	return time.Unix(int64(ut/1000), int64(ut%1000)*1000000)
}

const ExifDate = "2006:01:02 15:04:05.999"

// MarshalYAML is YAML marshaler interface implementation.
func (ut unix_t) MarshalYAML() (interface{}, error) {
	return ut.Time().Format(ExifDate), nil
}

// UnmarshalYAML is YAML unmarshaler interface implementation.
func (ut *unix_t) UnmarshalYAML(value *yaml.Node) (err error) {
	var t time.Time
	if t, err = time.Parse(ExifDate, value.Value); err != nil {
		return
	}
	*ut = UnixJS(t)
	return
}

// UnixJS converts time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) unix_t {
	return unix_t(u.UnixNano() / 1000000)
}

// UnixJSNow returns same result as Date.now() in javascript.
func UnixJSNow() unix_t {
	return unix_t(time.Now().UnixNano() / 1000000)
}

// TimeJS is backward conversion from javascript compatible Unix time
// in milliseconds to golang structure.
func TimeJS(ujs unix_t) time.Time {
	return time.Unix(int64(ujs/1000), int64(ujs%1000)*1000000)
}
