package hms

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	cfg "github.com/schwarzlichtbezirk/hms/config"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const (
	devmsuff = "devmode" // relative path to folder with development mode code files
	relmsuff = "build"   // relative path to folder with compiled code files
)

// HTTP distribution cache
var pagecache = map[string][]byte{}

// Pages aliases.
var pagealias = map[string]string{
	"/":     "main.html",
	"/stat": "stat.html",
	"/404":  "404.html",
}

// Routes aliases.
var routealias = map[string]string{
	"/fs/":   ".",
	"/devm/": devmsuff,
	"/relm/": relmsuff,
	"/plug/": "plugin",
	"/asst/": "assets",
}

// "Server" field for HTTP headers.
var serverlabel = fmt.Sprintf("hms/%s (%s)", cfg.BuildVers, runtime.GOOS)

var Offered = []string{
	binding.MIMEJSON,
	binding.MIMEXML,
	binding.MIMEYAML,
	binding.MIMETOML,
}

func Negotiate(c *gin.Context, code int, data any) {
	c.Header("Server", serverlabel)
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
		uid = uv.(*Profile).ID
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
	Log.Error("response error: %s", err.Error())
	RetErr(c, http.StatusInternalServerError, code, err)
}

// HdrRange describes one range chunk of the file to download.
type HdrRange struct {
	Start int64
	End   int64
}

// GetHdrRange returns array of ranges of file to download from request header.
func GetHdrRange(r *http.Request) (ret []HdrRange) {
	for _, hdr := range r.Header["Range"] {
		var chunks = strings.Split(strings.TrimPrefix(hdr, "bytes="), ", ")
		for _, chunk := range chunks {
			if vals := strings.Split(chunk, "-"); len(vals) == 2 {
				var rv HdrRange
				if vals[0] == "" {
					rv.Start = -1
				} else if i64, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
					rv.Start = i64
				}
				if vals[1] == "" {
					rv.End = -1
				} else if i64, err := strconv.ParseInt(vals[1], 10, 64); err == nil {
					rv.End = i64
				}
				ret = append(ret, rv)
			}
		}
	}
	return
}

// HasRangeBegin returns true if request headers have "Range" header
// with range thats starts from beginning of the file.
func HasRangeBegin(r *http.Request) bool {
	var ranges = GetHdrRange(r)
	if len(ranges) == 0 {
		return true
	}
	for _, rv := range ranges {
		if rv.Start == 0 {
			return true
		}
	}
	return false
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
	dacc.GET("/ctgr/:cat", SpiPage(devmsuff, pagealias["/"]))
	gacc.GET("/ctgr/:cat", SpiPage(relmsuff, pagealias["/"]))
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

	var api = r.Group("/api", ApiWrap)
	api.GET("/ping", SpiPing)
	api.POST("/reload", Auth(true), SpiReload)
	api.GET("/stat/srvinf", SpiServInfo)
	api.GET("/stat/memusg", SpiMemUsage)
	api.GET("/stat/cchinf", SpiCachesInfo)
	api.POST("/stat/getlog", SpiGetLog)
	api.POST("/stat/usrlst", SpiUserList)

	api.POST("/auth/signin", SpiSignin)
	api.GET("/auth/refresh", Auth(true), SpiRefresh)

	var usr = gacc.Group("/api")

	usr.POST("/res/folder", Auth(false), SpiFolder)
	usr.POST("/res/tags", Auth(false), SpiTags)
	usr.POST("/res/ispath", Auth(true), SpiHasPath)

	usr.POST("/gps/range", Auth(true), SpiGpsRange)
	usr.POST("/gps/scan", Auth(false), SpiGpsScan)

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

	usr.POST("/edit/copy", Auth(true), SpiEditCopy)
	usr.POST("/edit/rename", Auth(true), SpiEditRename)
	usr.POST("/edit/delete", Auth(true), SpiEditDelete)
}
