package hms

import (
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

const (
	// "iss" field for this tokens.
	jwtIssuer = "hms"

	// Pointer to Profile object stored at gin context
	// after successful authorization.
	userKey = "user"

	realmBasic  = `Basic realm="slotopol", charset="UTF-8"`
	realmBearer = `JWT realm="slotopol", charset="UTF-8"`
)

// HTTP error messages
var (
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoScheme = errors.New("authorization does not have expected scheme")
	ErrNoJwtID  = errors.New("jwt-token does not have user id")
	ErrBadJwtID = errors.New("jwt-token id does not refer to registered user")
	ErrNoCred   = errors.New("profile with given credentials does not registered")
	ErrNoPubKey = errors.New("public key does not exist any more")
	ErrNotPass  = errors.New("password is incorrect")
	ErrNoAcc    = errors.New("profile is absent")
)

// Claims of JWT-tokens. Contains additional profile identifier.
type Claims struct {
	jwt.RegisteredClaims
	UID ID_t `json:"uid,omitempty"`
}

func (c *Claims) Validate() error {
	if c.UID == 0 {
		return ErrNoJwtID
	}
	return nil
}

type AuthGetter func(c *gin.Context) (*Profile, int, error)

// AuthGetters is the list of functions to extract the authorization
// data from the parts of request. List and order in it can be changed.
var AuthGetters = []AuthGetter{
	UserFromHeader, UserFromQuery, UserFromCookie,
}

// Auth is authorization middleware, sets User object associated
// with authorization to gin context. `required` parameter tells
// to continue if authorization is absent.
func Auth(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		var code int
		var user *Profile
		for _, getter := range AuthGetters {
			if user, code, err = getter(c); err != nil {
				Ret401(c, code, err)
				return
			}
			if user != nil {
				break
			}
		}

		if user == nil {
			if q := c.Param("aid"); q != "" {
				var aid ID_t
				if aid, err = ParseID(q); err != nil {
					Ret400(c, SEC_token_badaid, err)
					return
				}
				var ip = net.ParseIP(c.RemoteIP())
				if InPasslist(ip) {
					var ok bool
					if user, ok = Profiles.Get(aid); !ok {
						Ret404(c, SEC_param_noacc, ErrNoAcc)
						return
					}
				}
			}
		}

		if user != nil {
			c.Set(userKey, user)
		} else if required {
			Ret401(c, SEC_auth_absent, ErrNoAuth)
			return
		}

		c.Next()
	}
}

func UserFromHeader(c *gin.Context) (*Profile, int, error) {
	if hdr := c.Request.Header.Get("Authorization"); hdr != "" {
		if strings.HasPrefix(hdr, "Basic ") {
			return GetBasicAuth(hdr[6:])
		} else if strings.HasPrefix(hdr, "Bearer ") {
			return GetBearerAuth(hdr[7:])
		} else {
			return nil, SEC_auth_scheme, ErrNoScheme
		}
	}
	return nil, 0, nil
}

func UserFromQuery(c *gin.Context) (*Profile, int, error) {
	if credentials := c.Query("cred"); credentials != "" {
		return GetBasicAuth(credentials)
	} else if tokenstr := c.Query("token"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	} else if tokenstr := c.Query("jwt"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	}
	return nil, 0, nil
}

func UserFromCookie(c *gin.Context) (*Profile, int, error) {
	if credentials, _ := c.Cookie("cred"); credentials != "" {
		return GetBasicAuth(credentials)
	} else if tokenstr, _ := c.Cookie("token"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	} else if tokenstr, _ := c.Cookie("jwt"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	}
	return nil, 0, nil
}

func UserFromForm(c *gin.Context) (*Profile, int, error) {
	if credentials := c.PostForm("cred"); credentials != "" {
		return GetBasicAuth(credentials)
	} else if tokenstr := c.PostForm("token"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	} else if tokenstr := c.PostForm("jwt"); tokenstr != "" {
		return GetBearerAuth(tokenstr)
	}
	return nil, 0, nil
}

func GetBasicAuth(credentials string) (user *Profile, code int, err error) {
	var decoded []byte
	if decoded, err = base64.RawURLEncoding.DecodeString(credentials); err != nil {
		return nil, SEC_basic_decode, err
	}
	var parts = strings.Split(B2S(decoded), ":")

	var login = parts[0]
	Profiles.Range(func(uid ID_t, u *Profile) bool {
		if u.Login != login {
			return true
		}
		user = u
		return false
	})
	if user == nil {
		err, code = ErrNoCred, SEC_basic_noacc
		return
	}
	if user.Password != parts[1] {
		err, code = ErrNotPass, SEC_basic_deny
		return
	}
	return
}

func GetBearerAuth(tokenstr string) (prf *Profile, code int, err error) {
	var claims Claims
	_, err = jwt.ParseWithClaims(tokenstr, &claims, func(*jwt.Token) (any, error) {
		var keys = jwt.VerificationKeySet{
			Keys: []jwt.VerificationKey{
				S2B(Cfg.AccessKey),
				S2B(Cfg.RefreshKey),
			},
		}
		return keys, nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuer(jwtIssuer), jwt.WithLeeway(5*time.Second))

	if err == nil {
		var ok bool
		if prf, ok = Profiles.Get(claims.UID); !ok {
			code, err = SEC_token_noacc, ErrBadJwtID
		}
		return
	}
	switch {
	case errors.Is(err, jwt.ErrTokenMalformed):
		code = SEC_token_malform
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		code = SEC_token_notsign
	case errors.Is(err, jwt.ErrTokenInvalidClaims):
		code = SEC_token_badclaims
	case errors.Is(err, jwt.ErrTokenExpired):
		code = SEC_token_expired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		code = SEC_token_notyet
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		code = SEC_token_issuer
	default:
		code = SEC_token_error
	}
	return
}

func GetUser(c *gin.Context) *Profile {
	if v, ok := c.Get(userKey); ok {
		return v.(*Profile)
	}
	return nil
}

func Handle404(c *gin.Context) {
	Ret404(c, SEC_nourl, Err404)
}

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
			Issuer:    jwtIssuer,
		},
		UID: uid,
	}).SignedString([]byte(Cfg.AccessKey))
	t.Refrsh, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  now,
			NotBefore: now,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(Cfg.RefreshTTL)),
			Issuer:    jwtIssuer,
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
func GetAuth(r *http.Request) (prf *Profile, code int, err error) {
	if hdr := r.Header.Get("Authorization"); hdr != "" {
		if strings.HasPrefix(hdr, "Basic ") {
			return GetBasicAuth(hdr[6:])
		} else if strings.HasPrefix(hdr, "Bearer ") {
			return GetBearerAuth(hdr[7:])
		} else {
			return nil, SEC_auth_scheme, ErrNoScheme
		}
	}

	var vars = mux.Vars(r)
	if vars == nil {
		return // no authorization
	}
	var aid ID_t
	if aid, err = ParseID(vars["aid"]); err != nil {
		code = SEC_token_badaid
		return // no authorization
	}
	var ip = net.ParseIP(StripPort(r.RemoteAddr))
	if InPasslist(ip) {
		var ok bool
		if prf, ok = Profiles.Get(aid); !ok {
			code, err = SEC_param_noacc, ErrNoAcc
			return
		}
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
		var prf *Profile
		var uid ID_t
		var code int
		if prf, code, err = GetAuth(r); err != nil {
			WriteRet(w, r, http.StatusUnauthorized, MakeAjaxErr(err, code))
			return
		} else if prf != nil {
			uid = prf.ID
		}

		fn(w, r, aid, uid)
	}
}

// The End.
