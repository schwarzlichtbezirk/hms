package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bluele/gcache"
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

// authentication settings
var (
	// Access token time to live.
	AccessTTL int
	// Refresh token time to live.
	RefreshTTL int
	// Key for access HS-256 JWT-tokens.
	AccessKey string
	// Key for refresh HS-256 JWT-tokens.
	RefreshKey string
	// Can list of all shares be visible for unauthorized user.
	ShowSharesUser bool
)

var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoBearer = errors.New("authorization does not have bearer format")
	ErrNoUserID = errors.New("token does not have user id")
	ErrNoPubKey = errors.New("public key does not exist any more")
	ErrBadPass  = errors.New("password is incorrect")
	ErrDeny     = errors.New("access denied")
	ErrNoAcc    = errors.New("account is absent")
)

// Public keys cache for authorization.
var pubkeycache = gcache.New(10).LRU().Expiration(15 * time.Second).Build()

// Zero hash value.
var zero [32]byte

type Tokens struct {
	Access string `json:"access"`
	Refrsh string `json:"refrsh"`
}

type AuthHandlerFunc func(w http.ResponseWriter, r *http.Request, auth *Account)

func (t *Tokens) Make(aid int) {
	var now = time.Now()
	t.Access, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &HMSClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Duration(AccessTTL) * time.Second).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &HMSClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Duration(RefreshTTL) * time.Second).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(RefreshKey))
}

func StripPort(hostport string) string {
	var colon = strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	if i := strings.IndexByte(hostport, ']'); i != -1 {
		return strings.TrimPrefix(hostport[:i], "[")
	}
	return hostport[:colon]
}

func IsLocalhost(host string) bool {
	host = StripPort(host)
	if host == "localhost" {
		return true
	}
	var ip = net.ParseIP(host)
	return ip.IsLoopback()
}

func CheckAuth(r *http.Request) (auth *Account, aerr error) {
	if pool, is := r.Header["Authorization"]; is {
		var err error // stores last bearer error
		var claims HMSClaims
		var bearer = false
		for _, val := range pool {
			if strings.HasPrefix(val, "Bearer ") {
				bearer = true
				if _, err = jwt.ParseWithClaims(val[7:], &claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(AccessKey), nil
				}); err != nil {
					break
				}
			}
		}
		if !bearer {
			aerr = &AjaxErr{ErrNoBearer, EC_tokenless}
			return
		} else if _, is := err.(*jwt.ValidationError); is {
			aerr = &AjaxErr{err, EC_tokenerror}
			return
		} else if err != nil {
			aerr = &AjaxErr{err, EC_tokenbad}
			return
		} else if auth = AccList.ByID(claims.AID); auth == nil {
			aerr = &AjaxErr{ErrNoAcc, EC_tokennoacc}
			return
		}
	} else if !IsLocalhost(r.Host) {
		aerr = &AjaxErr{ErrNoAuth, EC_noauth}
		return
	} else {
		auth = AccList.ByID(DefAccID)
	}
	return
}

// Handler wrapper for API with admin checkup.
func AuthWrap(fn AuthHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)

		var err error
		var auth *Account
		if auth, err = CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}

		fn(w, r, auth)
	}
}

func pubkeyApi(w http.ResponseWriter, r *http.Request) {
	var err error
	var buf [32]byte
	if _, err = randbytes(buf[:]); err != nil {
		WriteError500(w, err, EC_pubkeyrand)
		return
	}

	pubkeycache.Set(tohex(buf[:]), struct{}{})

	WriteJson(w, http.StatusOK, buf)
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
		if arg.Name == "" || bytes.Equal(arg.PubK[:], zero[:]) || bytes.Equal(arg.Hash[:], zero[:]) {
			WriteError400(w, ErrNoData, EC_signinnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_signinnoreq)
		return
	}

	var acc *Account
	if acc = AccList.ByLogin(arg.Name); acc == nil {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrNoAcc, EC_signinnoacc})
		return
	}

	if _, err = pubkeycache.Get(tohex(arg.PubK[:])); err != nil {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrNoPubKey, EC_signinpkey})
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write([]byte(acc.Password))
	var cmp = mac.Sum(nil)
	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrBadPass, EC_signindeny})
		return
	}

	res.Make(acc.ID)

	WriteJson(w, http.StatusOK, &res)
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
		return []byte(RefreshKey), nil
	}); err != nil {
		WriteError400(w, err, EC_refrshparse)
		return
	}

	res.Make(claims.AID)

	WriteJson(w, http.StatusOK, &res)
}

// The End.
