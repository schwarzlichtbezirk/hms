package hms

import (
	"encoding/base64"
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
		return ErrNoJwtID
	}
	if !Profiles.Has(c.UID) {
		return ErrBadJwtID
	}
	return nil
}

// HTTP error messages
var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoScheme = errors.New("authorization does not have expected scheme")
	ErrIssOut   = errors.New("outside jwt-token issuer")
	ErrNoJwtID  = errors.New("jwt-token does not have user id")
	ErrBadJwtID = errors.New("jwt-token id does not refer to registered user")
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

func GetBasicAuth(credentials string) (uid ID_t, aerr *AjaxErr) {
	var err error
	var decoded []byte
	if decoded, err = base64.RawURLEncoding.DecodeString(credentials); err != nil {
		aerr = MakeAjaxErr(err, SEC_basic_decode)
		return
	}
	var parts = strings.Split(B2S(decoded), ":")

	var prf *Profile
	if prf = ProfileByUser(parts[0]); prf == nil {
		aerr = MakeAjaxErr(ErrNoAcc, SEC_basic_noacc)
		return
	}

	if parts[1] != prf.Password {
		aerr = MakeAjaxErr(ErrBadPass, SEC_basic_deny)
		return
	}

	uid = prf.ID
	return
}

func GetBearerAuth(tokenstr string) (uid ID_t, aerr *AjaxErr) {
	var err error
	var claims Claims
	_, err = jwt.ParseWithClaims(tokenstr, &claims, func(*jwt.Token) (any, error) {
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
	}, jwt.WithLeeway(5*time.Second))

	if err == nil {
		uid = claims.UID
		return
	}
	switch {
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
}

// GetAuth returns profile ID from authorization header if it present,
// or default profile with no error if authorization is absent on localhost.
// Returns nil pointer and nil error on unauthorized request from any host.
func GetAuth(r *http.Request) (uid ID_t, aerr *AjaxErr) {
	var err error
	if hdr := r.Header.Get("Authorization"); hdr != "" {
		if strings.HasPrefix(hdr, "Basic ") {
			return GetBasicAuth(hdr[6:])
		} else if strings.HasPrefix(hdr, "Bearer ") {
			return GetBearerAuth(hdr[7:])
		} else {
			aerr = MakeAjaxErr(ErrNoScheme, SEC_auth_scheme)
			return
		}
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
	if !Profiles.Has(aid) {
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
		var vars = mux.Vars(r)
		if vars == nil {
			panic("bad route for URL " + r.URL.Path)
		}
		var aid ID_t
		var err error
		if aid, err = ParseID(vars["aid"]); err != nil {
			WriteError400(w, r, err, SEC_noaid)
			return
		}
		var uid ID_t
		var aerr *AjaxErr
		if uid, aerr = GetAuth(r); aerr != nil {
			WriteRet(w, r, http.StatusUnauthorized, aerr)
			return
		}

		fn(w, r, aid, uid)
	}
}

// The End.
