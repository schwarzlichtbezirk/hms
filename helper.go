package hms

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// API calls counters
	pagecallcount = map[string]uint64{}
	pccmux        sync.Mutex

	ajaxcallcount uint64
	ajaxcallmeter = Meter{
		CPtr: &ajaxcallcount,
	}
	foldercallcout uint64
	sharecallcount uint64
	localcallcount uint64

	// meters scanner
	meterscanner *time.Timer
)

var (
	randbytes  = rand.Read
	randfloat  = rand.Float64
	tohex      = hex.EncodeToString
	fromhex    = hex.DecodeString
	tobase64   = base64.StdEncoding.EncodeToString
	frombase64 = base64.StdEncoding.DecodeString
	incint     = atomic.AddInt64
	incuint    = atomic.AddUint64
	ldint      = atomic.LoadInt64
	lduint     = atomic.LoadUint64
)

func makehash(data []byte, password []byte) []byte {
	var h = sha256.New()
	h.Write(password)
	h.Write(data)
	return h.Sum(nil)
}

func UnixJS(u time.Time) int64 {
	return u.UnixNano() / 1000000
}

///////////
// Meter //
///////////

const metsize = 11

type Meter struct {
	CPtr *uint64
	prev [metsize]uint64
}

func (m *Meter) Update() {
	copy(m.prev[1:], m.prev[0:])
	m.prev[0] = lduint(m.CPtr)
}

func (m *Meter) Freq() float64 {
	if m.prev[0] == 0 {
		return 0
	}

	var f float64
	var i int
	for i = 0; i < metsize-1 && m.prev[i] > 0; i++ {
		f += float64(m.prev[i] - m.prev[i+1])
	}
	return f / float64(i)
}

func meterupdater() {
	meterscanner.Reset(time.Second)

	ajaxcallmeter.Update()
}

// The End.
