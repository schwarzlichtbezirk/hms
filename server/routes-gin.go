package hms

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

var Offered = []string{
	binding.MIMEJSON,
	binding.MIMEXML,
	binding.MIMEYAML,
	binding.MIMETOML,
}

func Negotiate(c *gin.Context, code int, data any) {
	switch c.NegotiateFormat(Offered...) {
	case binding.MIMEJSON:
		c.JSON(code, data)
	case binding.MIMEXML:
		c.XML(code, data)
	case binding.MIMEYAML:
		c.YAML(code, data)
	case binding.MIMETOML:
		c.TOML(code, data)
	default:
		c.JSON(code, data)
	}
	c.Abort()
}

func RetOk(c *gin.Context, data any) {
	Negotiate(c, http.StatusOK, data)
}

type jerr struct {
	error
}

// Unwrap returns inherited error object.
func (err jerr) Unwrap() error {
	return err.error
}

// MarshalJSON is standard JSON interface implementation to stream errors on Ajax.
func (err jerr) MarshalJSON() ([]byte, error) {
	return json.Marshal(err.Error())
}

// MarshalYAML is YAML marshaler interface implementation to stream errors on Ajax.
func (err jerr) MarshalYAML() (any, error) {
	return err.Error(), nil
}

// MarshalXML is XML marshaler interface implementation to stream errors on Ajax.
func (err jerr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(err.Error(), start)
}

type ajaxerr struct {
	XMLName xml.Name `json:"-" yaml:"-" xml:"error"`
	What    jerr     `json:"what" yaml:"what" xml:"what"`
	Code    int      `json:"code,omitempty" yaml:"code,omitempty" xml:"code,omitempty"`
	UID     uint64   `json:"uid,omitempty" yaml:"uid,omitempty" xml:"uid,omitempty,attr"`
}

func (err ajaxerr) Error() string {
	return fmt.Sprintf("what: %s, code: %d", err.What, err.Code)
}

func (err ajaxerr) Unwrap() error {
	return err.What.error
}

func RetErr(c *gin.Context, status, code int, err error) {
	var uid uint64
	if uv, ok := c.Get(userKey); ok {
		uid = uint64(uv.(*Profile).ID)
	}
	Negotiate(c, status, ajaxerr{
		What: jerr{err},
		Code: code,
		UID:  uid,
	})
}

func Ret400(c *gin.Context, code int, err error) {
	RetErr(c, http.StatusBadRequest, code, err)
}

func Ret401(c *gin.Context, code int, err error) {
	c.Writer.Header().Add("WWW-Authenticate", realmBasic)
	c.Writer.Header().Add("WWW-Authenticate", realmBearer)
	RetErr(c, http.StatusUnauthorized, code, err)
}

func Ret403(c *gin.Context, code int, err error) {
	RetErr(c, http.StatusForbidden, code, err)
}

func Ret404(c *gin.Context, code int, err error) {
	RetErr(c, http.StatusNotFound, code, err)
}

func Ret500(c *gin.Context, code int, err error) {
	RetErr(c, http.StatusInternalServerError, code, err)
}

func Router(r *gin.Engine) {
	r.NoRoute(Auth(false), Handle404)

	var rdev = r.Group("/dev")
	var dacc = rdev.Group("/id:aid")
	var gacc = r.Group("/id:aid")

	//////////////////////
	// content delivery //
	//////////////////////

	// wpk-files sharing
	var rgz = r.Group("/", gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{
		".avif", ".webp", ".jp2", ".jpg", ".png", ".gif", ".woff", ".woff2",
	})))
	for alias, prefix := range routealias {
		var sub, err = ResFS.Sub(prefix)
		if err != nil {
			Log.Fatal(err)
		}
		rgz.StaticFS(alias, http.FS(sub))
	}

	// UI pages
	for fpath, fname := range pagealias {
		rdev.GET(fpath, SpiPage(devmsuff, fname)) // development mode
		r.GET(fpath, SpiPage(relmsuff, fname))    // release mode
	}
	dacc.GET("/path/*path", SpiPage(devmsuff, pagealias["/"]))
	gacc.GET("/path/*path", SpiPage(relmsuff, pagealias["/"]))

	// file system sharing & converted media files
	gacc.GET("/file/*path", Auth(false), SpiFile)
	// embedded thumbnails
	gacc.GET("/etmb/:puid", Auth(false), SpiEtmb)
	// cached thumbnails
	gacc.GET("/mtmb/:puid", Auth(false), SpiMtmb)
	// cached tiles
	gacc.GET("/tile/:puid/:dim", Auth(false), SpiTile)

	////////////////
	// API routes //
	////////////////

	var api = r.Group("/api")
	api.GET("/ping", SpiPing)
	api.GET("/stat/srvinf", SpiServInfo)
	api.GET("/stat/memusg", SpiMemUsage)
	api.GET("/stat/cchinf", SpiCachesInfo)
	api.POST("/stat/getlog", SpiGetLog)
	api.POST("/stat/usrlst", SpiUserList)

	var usr = gacc.Group("/api")
	usr.POST("/tags/check", Auth(false), SpiTagsCheck)
	usr.POST("/tags/start", Auth(false), SpiTagsStart)
	usr.POST("/tags/break", Auth(false), SpiTagsBreak)
	usr.POST("/tile/check", Auth(false), SpiTileCheck)
	usr.POST("/tile/start", Auth(false), SpiTileStart)
	usr.POST("/tile/break", Auth(false), SpiTileBreak)

	usr.POST("/drive/add", Auth(true), SpiDriveAdd)
	usr.POST("/drive/del", Auth(true), SpiDriveDel)

	usr.POST("/cloud/add", Auth(true), SpiCloudAdd)
	usr.POST("/cloud/del", Auth(true), SpiCloudDel)

	usr.POST("/share/add", Auth(true), SpiShareAdd)
	usr.POST("/share/del", Auth(true), SpiShareDel)
}
