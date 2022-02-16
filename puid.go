package hms

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// IdType is the type of any users identifiers
type IdType uint64

// PuidType represents integer form of path unique ID.
type PuidType uint64

// Predefined PUIDs.
const (
	PUIDhome   PuidType = 1
	PUIDdrives PuidType = 2
	PUIDshares PuidType = 3
	PUIDmedia  PuidType = 4
	PUIDvideo  PuidType = 5
	PUIDaudio  PuidType = 6
	PUIDimage  PuidType = 7
	PUIDbooks  PuidType = 8
	PUIDtexts  PuidType = 9

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

var CatNames = map[PuidType]string{
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
var CatKeyPath = map[PuidType]string{
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
var CatPathKey = map[string]PuidType{
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
func (pt PuidType) String() string {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(pt))
	var n int
	for n = 7; n >= 0 && buf[n] == 0; n-- {
	}
	return idenc.EncodeToString(buf[:n+1])
}

// Set writes base32 string representation of ID into integer value.
func (pt *PuidType) Set(puid string) error {
	var buf [8]byte
	_, err := idenc.Decode(buf[:], []byte(puid))
	*pt = PuidType(binary.LittleEndian.Uint64(buf[:]))
	return err
}

// MarshalJSON is JSON marshaler interface implementation.
func (pt PuidType) MarshalJSON() ([]byte, error) {
	return json.Marshal(pt.String())
}

// UnmarshalJSON is JSON unmarshaler interface implementation.
func (pt *PuidType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return pt.Set(s)
}

// MarshalYAML is YAML marshaler interface implementation.
func (pt PuidType) MarshalYAML() (interface{}, error) {
	return pt.String(), nil
}

// UnmarshalYAML is YAML unmarshaler interface implementation.
func (pt *PuidType) UnmarshalYAML(value *yaml.Node) error {
	return pt.Set(value.Value)
}

// Rand generates random identifier of given length in bits, maximum 64 bits.
// Multiply x5 to convert length in base32 symbols to bits.
func (pt *PuidType) Rand(bits int) {
	var buf [8]byte
	if _, err := rand.Read(buf[:(bits+7)/8]); err != nil {
		panic(err)
	}
	*pt = PuidType(binary.LittleEndian.Uint64(buf[:]))
	*pt &= 0xffffffffffffffff >> (65 - bits) // throw one more bit to prevent string representation overflow
}
