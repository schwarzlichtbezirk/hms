package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
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

// Claims of JWT-tokens. Contains additional profile identifier.
type Claims struct {
	jwt.StandardClaims
	AID int `json:"aid"`
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

// Zero hash value.
var zero32 [32]byte

// Tokens is access and refresh tokens pair.
type Tokens struct {
	Access string `json:"access"`
	Refrsh string `json:"refrsh"`
}

// AuthHandlerFunc is type of handler for authorized API calls.
type AuthHandlerFunc func(w http.ResponseWriter, r *http.Request, auth *Profile)

// Make creates access and refresh tokens pair for given AID.
func (t *Tokens) Make(aid int) {
	var now = time.Now()
	t.Access, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(cfg.AccessTTL).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(cfg.AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(cfg.RefreshTTL).Unix(),
			Subject:   jwtsubject,
		},
		AID: aid,
	}).SignedString([]byte(cfg.RefreshKey))
}

// UnixJS converts time to UNIX-time in milliseconds, compatible with javascript time format.
func UnixJS(u time.Time) int64 {
	return u.UnixNano() / 1000000
}

// UnixJSNow returns same result as Date.now() in javascript.
func UnixJSNow() int64 {
	return time.Now().UnixNano() / 1000000
}

// TimeJS is backward conversion from javascript compatible Unix time
// in milliseconds to golang structure.
func TimeJS(ujs int64) time.Time {
	return time.Unix(ujs/1000, (ujs%1000)*1000000)
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

// IsLocalhost do check up host:port string refer is to localhost.
func IsLocalhost(addrport string) bool {
	return net.ParseIP(StripPort(addrport)).IsLoopback()
}

// GetAuth returns profile from authorization header if it present,
// or default profile with no error if authorization is absent.
func GetAuth(r *http.Request) (auth *Profile, aerr error) {
	defer func() {
		if auth != nil {
			go func() { usermsg <- UsrMsg{r, "auth", auth.ID} }()
		} else {
			go func() { usermsg <- UsrMsg{r, "auth", 0} }()
		}
	}()
	if pool, is := r.Header["Authorization"]; is {
		var err error // stores last bearer error
		var claims Claims
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
			aerr = &ErrAjax{ErrNoBearer, AECtokenless}
			return
		} else if _, is := err.(*jwt.ValidationError); is {
			aerr = &ErrAjax{err, AECtokenerror}
			return
		} else if err != nil {
			aerr = &ErrAjax{err, AECtokenbad}
			return
		} else if auth = prflist.ByID(claims.AID); auth == nil {
			aerr = &ErrAjax{ErrNoAcc, AECtokennoacc}
			return
		}
		return
	}
	if IsLocalhost(r.RemoteAddr) {
		auth = prflist.ByID(cfg.DefAccID)
	}
	return
}

// AuthWrap is handler wrapper for API with admin checkup.
func AuthWrap(fn AuthHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			userajax <- r
		}()

		var err error
		var auth *Profile
		if auth, err = GetAuth(r); err != nil {
			WriteJSON(w, http.StatusUnauthorized, err)
			return
		}
		if auth == nil {
			WriteError(w, http.StatusUnauthorized, ErrNoAuth, AECnoauth)
			return
		}

		// lock before exit check
		handwg.Add(1)
		defer handwg.Done()

		// check on exit during handler is called
		select {
		case <-exitctx.Done():
			return
		default:
		}

		fn(w, r, auth)
	}
}

func pubkeyAPI(w http.ResponseWriter, _ *http.Request) {
	var err error
	var buf [32]byte
	if _, err = rand.Read(buf[:]); err != nil {
		WriteError500(w, err, AECpubkeyrand)
		return
	}

	pubkeycache.Set(idenc.EncodeToString(buf[:]), struct{}{})

	WriteOK(w, buf)
}

func signinAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg struct {
		Name string   `json:"name"`
		PubK [32]byte `json:"pubk"`
		Hash [32]byte `json:"hash"`
	}
	var res Tokens

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if arg.Name == "" || bytes.Equal(arg.PubK[:], zero32[:]) || bytes.Equal(arg.Hash[:], zero32[:]) {
		WriteError400(w, ErrNoData, AECsigninnodata)
		return
	}

	var prf *Profile
	if prf = prflist.ByLogin(arg.Name); prf == nil {
		WriteError(w, http.StatusForbidden, ErrNoAcc, AECsigninnoacc)
		return
	}

	if _, err = pubkeycache.Get(idenc.EncodeToString(arg.PubK[:])); err != nil {
		WriteError(w, http.StatusForbidden, ErrNoPubKey, AECsigninpkey)
		return
	}

	var mac = hmac.New(sha256.New, arg.PubK[:])
	mac.Write([]byte(prf.Password))
	var cmp = mac.Sum(nil)
	if !hmac.Equal(arg.Hash[:], cmp) {
		WriteError(w, http.StatusForbidden, ErrBadPass, AECsignindeny)
		return
	}

	res.Make(prf.ID)

	WriteOK(w, &res)
}

func refrshAPI(w http.ResponseWriter, r *http.Request) {
	var err error
	var arg Tokens
	var res Tokens

	// get arguments
	if err = AjaxGetArg(w, r, &arg); err != nil {
		return
	}
	if len(arg.Refrsh) == 0 {
		WriteError400(w, ErrNoData, AECrefrshnodata)
		return
	}

	var claims Claims
	if _, err = jwt.ParseWithClaims(arg.Refrsh, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.RefreshKey), nil
	}); err != nil {
		WriteError400(w, err, AECrefrshparse)
		return
	}

	res.Make(claims.AID)

	WriteOK(w, &res)
}

// The End.
