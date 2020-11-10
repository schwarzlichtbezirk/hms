package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// jwt-go docs:
// https://godoc.org/github.com/dgrijalva/jwt-go

// "sub" field for this tokens.
const jwtsubject = "hms"

// Claims of JWT-tokens. Contains additional account identifier.
type HMSClaims struct {
	jwt.StandardClaims
	AID int `json:"aid"`
}

// Authentication settings.
type CfgAuth struct {
	// Access token time to live.
	AccessTTL int `json:"access-ttl" yaml:"access-ttl"`
	// Refresh token time to live.
	RefreshTTL int `json:"refresh-ttl" yaml:"refresh-ttl"`
	// Key for access HS-256 JWT-tokens.
	AccessKey string `json:"access-key" yaml:"access-key"`
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string `json:"refresh-key" yaml:"refresh-key"`
}

var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoBearer = errors.New("authorization does not have bearer format")
	ErrNoUserID = errors.New("token does not have user id")
	ErrNoPubKey = errors.New("public key does not exist any more")
	ErrBadPass  = errors.New("password is incorrect")
	ErrNoAcc    = errors.New("account is absent")
)

// Zero hash value.
var zero32 [32]byte

// Access and refresh tokens pair.
type Tokens struct {
	Access string `json:"access"`
	Refrsh string `json:"refrsh"`
}

// Type of handler for authorized API calls.
type AuthHandlerFunc func(w http.ResponseWriter, r *http.Request, auth *Account)

// Creates access and refresh tokens pair for given AID.
func (t *Tokens) Make(aid int) {
	var now = time.Now()
	t.Access, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &HMSClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Duration(cfg.AccessTTL) * time.Second).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(cfg.AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &HMSClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Duration(cfg.RefreshTTL) * time.Second).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(cfg.RefreshKey))
}

// Convert time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) int64 {
	return u.UnixNano() / 1000000
}

// Returns same result as Date.now() in javascript.
func UnixJSNow() int64 {
	return time.Now().UnixNano() / 1000000
}

// Fast IP-address extract from valid host:port string.
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

// Check up host:port string refer is to localhost.
func IsLocalhost(addrport string) bool {
	return net.ParseIP(StripPort(addrport)).IsLoopback()
}

// Returns account from authorization header if it present,
// or default account with no error if authorization is absent.
func GetAuth(r *http.Request) (auth *Account, aerr error) {
	if pool, is := r.Header["Authorization"]; is {
		var err error // stores last bearer error
		var claims HMSClaims
		var bearer = false
		for _, val := range pool {
			if strings.HasPrefix(val, "Bearer ") {
				bearer = true
				if _, err = jwt.ParseWithClaims(val[7:], &claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(cfg.AccessKey), nil
				}); err != nil {
					break
				}
			}
		}
		if !bearer {
			aerr = &ErrAjax{ErrNoBearer, EC_tokenless}
			return
		} else if _, is := err.(*jwt.ValidationError); is {
			aerr = &ErrAjax{err, EC_tokenerror}
			return
		} else if err != nil {
			aerr = &ErrAjax{err, EC_tokenbad}
			return
		} else if auth = acclist.ByID(claims.AID); auth == nil {
			aerr = &ErrAjax{ErrNoAcc, EC_tokennoacc}
			return
		}
		return
	}
	if IsLocalhost(r.RemoteAddr) {
		auth = acclist.ByID(cfg.DefAccID)
	}
	return
}

// Handler wrapper for API with admin checkup.
func AuthWrap(fn AuthHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userajax <- r

		var err error
		var auth *Account
		if auth, err = GetAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}
		if auth == nil {
			WriteError(w, http.StatusUnauthorized, ErrNoAuth, EC_noauth)
			return
		}

		fn(w, r, auth)
	}
}

func pubkeyApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var buf [32]byte
	if _, err = rand.Read(buf[:]); err != nil {
		WriteError500(w, err, EC_pubkeyrand)
		return
	}

	pubkeycache.Set(idenc.EncodeToString(buf[:]), struct{}{})

	WriteOK(w, buf)
}

func signinApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Name string   `json:"name"`
		PubK [32]byte `json:"pubk"`
		Hash [32]byte `json:"hash"`
	}
	var res Tokens

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_signinbadreq)
			return
		}
		if arg.Name == "" || bytes.Equal(arg.PubK[:], zero32[:]) || bytes.Equal(arg.Hash[:], zero32[:]) {
			WriteError400(w, ErrNoData, EC_signinnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_signinnoreq)
		return
	}

	var acc *Account
	if acc = acclist.ByLogin(arg.Name); acc == nil {
		WriteError(w, http.StatusForbidden, ErrNoAcc, EC_signinnoacc)
		return
	}

	if _, err = pubkeycache.Get(idenc.EncodeToString(arg.PubK[:])); err != nil {
		WriteError(w, http.StatusForbidden, ErrNoPubKey, EC_signinpkey)
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write([]byte(acc.Password))
	var cmp = mac.Sum(nil)
	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteError(w, http.StatusForbidden, ErrBadPass, EC_signindeny)
		return
	}

	res.Make(acc.ID)

	WriteOK(w, &res)
}

func refrshApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg Tokens
	var res Tokens

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_refrshbadreq)
			return
		}
		if len(arg.Refrsh) == 0 {
			WriteError400(w, ErrNoData, EC_refrshnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_refrshnoreq)
		return
	}

	var claims HMSClaims
	if _, err = jwt.ParseWithClaims(arg.Refrsh, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.RefreshKey), nil
	}); err != nil {
		WriteError400(w, err, EC_refrshparse)
		return
	}

	res.Make(claims.AID)

	WriteOK(w, &res)
}

// The End.
