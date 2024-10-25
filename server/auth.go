package hms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// "iss" field for this tokens.
	jwtIssuer = "hms"

	// Pointer to Profile object stored at gin context
	// after successful authorization.
	userKey = "user"

	realmBasic  = `Basic realm="hms", charset="UTF-8"`
	realmBearer = `JWT realm="hms", charset="UTF-8"`
)

// HTTP error messages
var (
	ErrNoJwtID  = errors.New("jwt-token does not have user id")
	ErrBadJwtID = errors.New("jwt-token id does not refer to registered user")
	ErrNoAuth   = errors.New("authorization is absent")
	ErrNoScheme = errors.New("authorization does not have expected scheme")
	ErrNoSecret = errors.New("expected password or SHA25 hash on it and current time as a nonce")
	ErrSmallKey = errors.New("password too small")
	ErrNoCred   = errors.New("profile with given credentials does not registered")
	ErrNotPass  = errors.New("password is incorrect")
	ErrSigTime  = errors.New("signing time can not been recognized (time in RFC3339 expected)")
	ErrSigOut   = errors.New("nonce is expired")
	ErrBadHash  = errors.New("hash cannot be decoded in hexadecimal")
	ErrNoAcc    = errors.New("profile is absent")
)

// Claims of JWT-tokens. Contains additional profile identifier.
type Claims struct {
	jwt.RegisteredClaims
	UID uint64 `json:"uid,omitempty"`
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
			if s := c.Param("aid"); s != "" {
				var aid uint64
				if aid, err = strconv.ParseUint(s, 10, 64); err != nil {
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
	Profiles.Range(func(uid uint64, u *Profile) bool {
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

func Handle404(c *gin.Context) {
	if strings.HasPrefix(c.Request.RequestURI, "/api/") {
		Ret404(c, SEC_nourl, Err404)
		return
	}
	var content = pagecache[devmsuff+"/404.html"]
	c.Header("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(c.Writer, c.Request, "404.html", starttime, bytes.NewReader(content))
}

func Handle405(c *gin.Context) {
	if strings.HasPrefix(c.Request.RequestURI, "/api/") {
		RetErr(c, http.StatusMethodNotAllowed, SEC_nomethod, Err405)
		return
	}
	var content = pagecache[devmsuff+"/404.html"]
	c.Header("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(c.Writer, c.Request, "404.html", starttime, bytes.NewReader(content))
}

type AuthResp struct {
	XMLName xml.Name `json:"-" yaml:"-" xml:"ret"`
	UID     uint64   `json:"uid" yaml:"uid" xml:"uid"`
	Access  string   `json:"access" yaml:"access" xml:"access"`
	Refrsh  string   `json:"refrsh" yaml:"refrsh" xml:"refrsh"`
	Expire  string   `json:"expire" yaml:"expire" xml:"expire"`
	Living  string   `json:"living" yaml:"living" xml:"living"`
}

func (r *AuthResp) Setup(user *Profile) {
	var err error
	var token *jwt.Token
	var now = jwt.NewNumericDate(time.Now())
	var exp = jwt.NewNumericDate(time.Now().Add(Cfg.AccessTTL))
	var age = jwt.NewNumericDate(time.Now().Add(Cfg.RefreshTTL))
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: now,
			ExpiresAt: exp,
			Issuer:    jwtIssuer,
		},
		UID: user.ID,
	})
	if r.Access, err = token.SignedString([]byte(Cfg.AccessKey)); err != nil {
		panic(err)
	}
	r.Expire = exp.Format(time.RFC3339)
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: now,
			ExpiresAt: age,
			Issuer:    jwtIssuer,
		},
		UID: user.ID,
	})
	if r.Refrsh, err = token.SignedString([]byte(Cfg.AccessKey)); err != nil {
		panic(err)
	}
	r.Living = age.Format(time.RFC3339)
	r.UID = user.ID
}

func SpiSignin(c *gin.Context) {
	var err error
	var arg struct {
		XMLName xml.Name `json:"-" yaml:"-" xml:"arg"`
		Login   string   `json:"login" yaml:"login" xml:"login" form:"login" binding:"required"`
		Secret  string   `json:"secret" yaml:"secret,omitempty" xml:"secret,omitempty" form:"secret"`
		HS256   string   `json:"hs256,omitempty" yaml:"hs256,omitempty" xml:"hs256,omitempty" form:"hs256"`
		SigTime string   `json:"sigtime,omitempty" yaml:"sigtime,omitempty" xml:"sigtime,omitempty" form:"sigtime"`
	}
	var ret AuthResp

	if err = c.ShouldBind(&arg); err != nil {
		Ret400(c, SEC_signin_nobind, err)
		return
	}
	if len(arg.SigTime) == 0 && len(arg.Secret) == 0 {
		Ret400(c, SEC_signin_nosecret, ErrNoSecret)
		return
	}
	if len(arg.Secret) > 0 && len(arg.Secret) < 6 {
		Ret400(c, SEC_signin_smallsec, ErrSmallKey)
		return
	}

	var user *Profile
	Profiles.Range(func(uid uint64, u *Profile) bool {
		if u.Login != arg.Login {
			return true
		}
		user = u
		return false
	})
	if user == nil {
		Ret403(c, SEC_signin_nouser, ErrNoCred)
		return
	}

	if len(arg.Secret) > 0 {
		if arg.Secret != user.Password {
			Ret403(c, SEC_signin_denypass, ErrNotPass)
			return
		}
	} else {
		var sigtime time.Time
		if sigtime, err = time.Parse(time.RFC3339, arg.SigTime); err != nil {
			Ret400(c, SEC_signin_sigtime, ErrSigTime)
			return
		}
		if time.Since(sigtime) > Cfg.NonceTimeout {
			Ret403(c, SEC_signin_timeout, ErrSigOut)
			return
		}

		var hs256 []byte
		if hs256, err = hex.DecodeString(arg.HS256); err != nil {
			Ret400(c, SEC_signin_hs256, ErrBadHash)
			return
		}

		var h = hmac.New(sha256.New, S2B(arg.SigTime))
		h.Write(S2B(user.Password))
		var master = h.Sum(nil)
		if !hmac.Equal(master, hs256) {
			Ret403(c, SEC_signin_denyhash, ErrNotPass)
			return
		}
	}

	ret.Setup(user)
	RetOk(c, ret)
}

func SpiRefresh(c *gin.Context) {
	var ret AuthResp

	var user = c.MustGet(userKey).(*Profile)
	ret.Setup(user)
	RetOk(c, ret)
}

func GetUID(c *gin.Context) uint64 {
	if v, ok := c.Get(userKey); ok {
		return v.(*Profile).ID
	}
	return 0
}

func GetUser(c *gin.Context) *Profile {
	if v, ok := c.Get(userKey); ok {
		return v.(*Profile)
	}
	return nil
}

func GetAID(c *gin.Context) (id uint64, err error) {
	id, err = strconv.ParseUint(c.Param("aid"), 10, 64)
	return
}

// GetUAID extract user agent ID from cookie.
func GetUAID(r *http.Request) (uaid uint64, err error) {
	var c *http.Cookie
	if c, err = r.Cookie("UAID"); err != nil {
		return
	}
	if uaid, err = strconv.ParseUint(c.Value, 10, 64); err != nil {
		return
	}
	return
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

var Passlist []net.IPNet

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

// The End.
