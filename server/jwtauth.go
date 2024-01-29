package hms

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// "iss" field for this tokens.
const jwtissuer = "hms"

// Claims of JWT-tokens. Contains additional profile identifier.
type Claims struct {
	jwt.RegisteredClaims
	UID ID_t `json:"uid,omitempty"`
}

func (c *Claims) Validate() error {
	if c.UID == 0 {
		return ErrNoUserID
	}
	if !HasProfile(c.UID) {
		return ErrBadUserID
	}
	return nil
}

// HTTP error messages
var (
	ErrNoAuth    = errors.New("authorization is absent")
	ErrNoBearer  = errors.New("authorization does not have bearer format")
	ErrIssOut    = errors.New("outside jwt-token issuer")
	ErrNoUserID  = errors.New("jwt-token does not have user id")
	ErrBadUserID = errors.New("jwt-token id does not refer to registered user")
	ErrNoPubKey  = errors.New("public key does not exist any more")
	ErrBadPass   = errors.New("password is incorrect")
	ErrNoAcc     = errors.New("profile is absent")
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
	var now = jwt.NewNumericDate(time.Now())
	t.Access, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  now,
			NotBefore: now,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(Cfg.AccessTTL)),
			Issuer:    jwtissuer,
		},
		UID: uid,
	}).SignedString([]byte(Cfg.AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  now,
			NotBefore: now,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(Cfg.RefreshTTL)),
			Issuer:    jwtissuer,
		},
		UID: uid,
	}).SignedString([]byte(Cfg.RefreshKey))
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
			if strings.HasPrefix(ToLower(val), "bearer ") {
				bearer = true
				if _, err = jwt.ParseWithClaims(val[7:], &claims, func(*jwt.Token) (any, error) {
					if claims.Issuer != jwtissuer {
						return nil, ErrIssOut
					}
					var keys = jwt.VerificationKeySet{
						Keys: []jwt.VerificationKey{
							S2B(Cfg.AccessKey),
							S2B(Cfg.RefreshKey),
						},
					}
					return keys, nil
				}, jwt.WithLeeway(5*time.Second)); err == nil {
					break // found valid authorization
				}
			}
		}
		switch {
		case !bearer:
			aerr = MakeAjaxErr(ErrNoBearer, SEC_token_less)
			return
		case err == nil:
			break
		case errors.Is(err, jwt.ErrTokenMalformed):
			aerr = MakeAjaxErr(err, SEC_token_malform)
			return
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			aerr = MakeAjaxErr(err, SEC_token_notsign)
			return
		case errors.Is(err, jwt.ErrTokenExpired):
			aerr = MakeAjaxErr(err, SEC_token_expired)
			return
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			aerr = MakeAjaxErr(err, SEC_token_notyet)
			return
		default:
			aerr = MakeAjaxErr(err, SEC_token_error)
			return
		}
		if err = claims.Validate(); err != nil {
			aerr = MakeAjaxErr(ErrNoAcc, SEC_token_noacc)
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
		aerr = MakeAjaxErr(err, SEC_token_badaid)
		return // no authorization
	}
	if !HasProfile(aid) {
		aerr = MakeAjaxErr(ErrNoAcc, SEC_token_noaid)
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
			WriteError400(w, r, err, SEC_noaid)
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

// The End.
