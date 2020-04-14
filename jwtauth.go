package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
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

// Key for HS-256 JWT-tokens.
const authkey = "skJgM4NsbP3fs4k7vh0gfdkgGl8dJTszdLxZ1sQ9ksFnxbgvw2RsGH8xxddUV479"

// Key for JWT ID.
const jwtidkey = "zxK4dUnuq3Lhd1Gzhpr3usI5lAzgvy2t3fmxld2spzz7a5nfv0hsksm9cheyutie"

// Claims of JWT-tokens. Contains additional user identifier.
type HMSClaims struct {
	UID uint64 `json:"user_id,omitempty"`
	jwt.StandardClaims
}

var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoBearer = errors.New("authorization does not have bearer format")
	ErrNoUserID = errors.New("token does not have user id")
	ErrNoPKey   = errors.New("public key does not exist any more")
	ErrNotPass  = errors.New("password is incorrect")
	ErrDeny     = errors.New("access denied")
)

// Zero hash
var zero [32]byte

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

func CheckAuth(r *http.Request) (aerr *AjaxErr) {
	var claims *HMSClaims
	if pool, ok := r.Header["Authorization"]; ok {
		var err error // stores last bearer error
		var token *jwt.Token
		for _, val := range pool {
			if strings.HasPrefix(val, "Bearer ") {
				claims = &HMSClaims{}
				if token, err = jwt.ParseWithClaims(val[7:], claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(authkey), nil
				}); token.Valid {
					break
				}
			}
		}
		if claims == nil {
			aerr = &AjaxErr{ErrNoBearer, EC_tokenless}
			return
		} else if _, is := err.(*jwt.ValidationError); is {
			aerr = &AjaxErr{err, EC_tokenerror}
			return
		} else if err != nil {
			aerr = &AjaxErr{err, EC_tokenbad}
			return
		}
	} else if !IsLocalhost(r.Host) {
		aerr = &AjaxErr{ErrNoAuth, EC_noauth}
	}
	return
}

// Handler wrapper for API with admin checkup.
func AuthWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		incuint(&ajaxcallcount, 1)

		if err := CheckAuth(r); err != nil {
			WriteJson(w, http.StatusUnauthorized, err)
			return
		}

		fn(w, r)
	}
}

func makeJWTID(t time.Time) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(UnixJS(t)))
	var h = md5.New()
	h.Write([]byte(jwtidkey))
	h.Write(buf[:])
	return h.Sum(nil)
}

var pubkeycache = gcache.New(10).LRU().Expiration(15 * time.Second).Build()

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
		PubK [32]byte `json:"pubk"`
		Hash [32]byte `json:"hash"`
	}
	var res struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}

	// get arguments
	if jb, _ := ioutil.ReadAll(r.Body); len(jb) > 0 {
		if err = json.Unmarshal(jb, &arg); err != nil {
			WriteError400(w, err, EC_signinbadreq)
			return
		}
		if bytes.Equal(arg.PubK[:], zero[:]) || bytes.Equal(arg.Hash[:], zero[:]) {
			WriteError400(w, ErrNoData, EC_signinnodata)
			return
		}
	} else {
		WriteError400(w, ErrNoJson, EC_signinnoreq)
		return
	}

	if _, err = pubkeycache.Get(tohex(arg.PubK[:])); err != nil {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrNoPKey, EC_signinpkey})
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write([]byte(AuthPass))
	var cmp = mac.Sum(nil)

	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteJson(w, http.StatusForbidden, &AjaxErr{ErrNotPass, EC_signindeny})
		return
	}

	WriteJson(w, http.StatusOK, &res)
}

// The End.
