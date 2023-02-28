package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

// jwt-go docs:
// https://godoc.org/github.com/dgrijalva/jwt-go

// "sub" field for this tokens.
const jwtsubject = "hms"

// Claims of JWT-tokens. Contains additional profile identifier.
type Claims struct {
	jwt.StandardClaims
	UID ID_t `json:"uid,omitempty"`
}

// HTTP error messages
var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoBearer = errors.New("authorization does not have bearer format")
	ErrNoUserID = errors.New("token does not have user id")
	ErrNoPubKey = errors.New("public key does not exist any more")
	ErrBadPass  = errors.New("password is incorrect")
	ErrNoAcc    = errors.New("profile is absent")
)

var Passlist []net.IPNet

// Zero hash value.
var zero32 [32]byte

// Tokens is access and refresh tokens pair.
type Tokens struct {
	Access string `json:"access" yaml:"access" xml:"access"`
	Refrsh string `json:"refrsh" yaml:"refrsh" xml:"refrsh"`
}

// AuthHandlerFunc is type of handler for authorized API calls.
type AuthHandlerFunc func(w http.ResponseWriter, r *http.Request, aid, uid ID_t)

// Make creates access and refresh tokens pair for given AID.
func (t *Tokens) Make(uid ID_t) {
	var now = time.Now()
	t.Access, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(cfg.AccessTTL).Unix(),
			Subject:   jwtsubject,
		},
		UID: uid,
	}).SignedString([]byte(cfg.AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(cfg.RefreshTTL).Unix(),
			Subject:   jwtsubject,
		},
		UID: uid,
	}).SignedString([]byte(cfg.RefreshKey))
}

// StripPort makes fast IP-address extract from valid host:port string.
func StripPort(addrport string) string {
	// IPv6
	if pos := strings.IndexByte(addrport, ']'); pos != -1 {
		return addrport[1:pos] // trim first '[' and after ']'
	}
	// IPv4
	if pos := strings.IndexByte(addrport, ':'); pos != -1 {
		return addrport[:pos]
	}
	return addrport // return as is otherwise
}

// ParseID is like ParseUint but for identifiers.
func ParseID(s string) (id ID_t, err error) {
	var u64 uint64
	if u64, err = strconv.ParseUint(s, 10, 64); err != nil {
		return
	}
	id = ID_t(u64)
	return
}

// GetUAID extract user agent ID from cookie.
func GetUAID(r *http.Request) (uaid ID_t, err error) {
	var c *http.Cookie
	if c, err = r.Cookie("UAID"); err != nil {
		return
	}
	if uaid, err = ParseID(c.Value); err != nil {
		return
	}
	return
}

// GetAuth returns profile ID from authorization header if it present,
// or default profile with no error if authorization is absent on localhost.
// Returns nil pointer and nil error on unauthorized request from any host.
func GetAuth(r *http.Request) (uid ID_t, aerr error) {
	var err error
	if pool, is := r.Header["Authorization"]; is {
		var claims Claims
		var bearer = false
		// at least one authorization field should have valid bearer access
		for _, val := range pool {
			if strings.HasPrefix(strings.ToLower(val), "bearer ") {
				bearer = true
				if _, err = jwt.ParseWithClaims(val[7:], &claims, func(*jwt.Token) (any, error) {
					if claims.UID > 0 {
						return s2b(cfg.AccessKey), nil
					} else {
						return nil, ErrNoUserID
					}
				}); err == nil {
					break // found valid authorization
				}
			}
		}
		var ve jwt.ValidationError
		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				aerr = MakeAjaxErr(err, AECtokenmalform)
				return
			} else if ve.Errors&jwt.ValidationErrorSignatureInvalid != 0 {
				aerr = MakeAjaxErr(err, AECtokennotsign)
				return
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				aerr = MakeAjaxErr(err, AECtokenexpired)
				return
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				aerr = MakeAjaxErr(err, AECtokennotyet)
				return
			} else {
				aerr = MakeAjaxErr(err, AECtokenerror)
				return
			}
		} else if err != nil {
			aerr = MakeAjaxErr(err, AECtokenerror)
			return
		} else if !bearer {
			aerr = MakeAjaxErr(ErrNoBearer, AECtokenless)
			return
		} else if prflist.ByID(claims.UID) == nil {
			aerr = MakeAjaxErr(ErrNoAcc, AECtokennoacc)
			return
		}
		uid = claims.UID
		return
	}

	var vars = mux.Vars(r)
	if vars == nil {
		return // no authorization
	}
	var aid ID_t
	if aid, err = ParseID(vars["aid"]); err != nil {
		return // no authorization
	}
	if prflist.ByID(aid) == nil {
		aerr = MakeAjaxErr(ErrNoAcc, AECtokennoaid)
		return
	}
	var ip = net.ParseIP(StripPort(r.RemoteAddr))
	if InPasslist(ip) {
		uid = aid
	}
	return
}

// InPasslist checks that IP is loopback or in passlist.
func InPasslist(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	for _, ipn := range Passlist {
		if ipn.Contains(ip) {
			return true
		}
	}
	return false
}

// AuthWrap is handler wrapper for API with admin checkup.
func AuthWrap(fn AuthHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var vars = mux.Vars(r)
		if vars == nil {
			panic("bad route for URL " + r.URL.Path)
		}
		var aid ID_t
		if aid, err = ParseID(vars["aid"]); err != nil {
			WriteError400(w, r, err, AECnoaid)
			return
		}
		var uid ID_t
		if uid, err = GetAuth(r); err != nil {
			WriteRet(w, r, http.StatusUnauthorized, err)
			return
		}

		fn(w, r, aid, uid)
	}
}

func pubkeyAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var ret struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`

		Key [32]byte `json:"key" yaml:"key,flow" xml:"key"`
	}
	if _, err = rand.Read(ret.Key[:]); err != nil {
		WriteError500(w, r, err, AECpubkeyrand)
		return
	}

	var cell TempCell[struct{}]
	cell.Data = nil
	cell.Wait = time.AfterFunc(15*time.Second, func() {
		pubkcache.Remove(ret.Key)
	})
	pubkcache.Set(ret.Key, cell)

	WriteOK(w, r, &ret)
}

func signinAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`

		Name string   `json:"name" yaml:"name" xml:"name"`
		PubK [32]byte `json:"pubk" yaml:"pubk" xml:"pubk"`
		Hash [32]byte `json:"hash" yaml:"hash" xml:"hash"`
	}
	var res Tokens

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if arg.Name == "" || bytes.Equal(arg.PubK[:], zero32[:]) || bytes.Equal(arg.Hash[:], zero32[:]) {
		WriteError400(w, r, ErrNoData, AECsigninnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByLogin(arg.Name); prf == nil {
		WriteError(w, r, http.StatusForbidden, ErrNoAcc, AECsigninnoacc)
		return
	}

	if _, ok := pubkcache.Get(arg.PubK); !ok {
		WriteError(w, r, http.StatusForbidden, ErrNoPubKey, AECsigninpkey)
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write(s2b(prf.Password))
	var cmp = mac.Sum(nil)
	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteError(w, r, http.StatusForbidden, ErrBadPass, AECsignindeny)
		return
	}

	pubkcache.Remove(arg.PubK) // remove public key on successful login

	res.Make(prf.ID)

	WriteOK(w, r, &res)
}

func refrshAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg Tokens
	var res Tokens

	// get arguments
	if err = ParseBody(w, r, &arg); err != nil {
		return
	}
	if len(arg.Refrsh) == 0 {
		WriteError400(w, r, ErrNoData, AECrefrshnodata)
		return
	}

	var claims Claims
	if _, err = jwt.ParseWithClaims(arg.Refrsh, &claims, func(token *jwt.Token) (any, error) {
		return s2b(cfg.RefreshKey), nil
	}); err != nil {
		WriteError400(w, r, err, AECrefrshparse)
		return
	}

	res.Make(claims.UID)

	WriteOK(w, r, &res)
}

// The End.
