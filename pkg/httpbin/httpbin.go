package httpbin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexa/pkg/ctx"
	"go.uber.org/zap"
)

// Default configuration values
const (
	DefaultMaxBodySize int64 = 1024 * 1024
	DefaultMaxDuration       = 10 * time.Second
	DefaultHostname          = "go-httpbin"
)

type statusCase struct {
	headers map[string]string
	body    []byte
}

type headersProcessorFunc func(h http.Header) http.Header

type HttpBin struct {
	ctx    *ctx.Ctx
	logger *zap.Logger
	g      *gin.Engine

	// The hostname to expose via /hostname.
	hostname string

	// Optional prefix under which the app will be served
	prefix string

	// If true, endpoints that allow clients to specify a response
	// Conntent-Type will NOT escape HTML entities in the response body, which
	// can enable (e.g.) reflected XSS attacks.
	//
	// This configuration is only supported for backwards compatibility if
	// absolutely necessary.
	unsafeAllowDangerousResponses bool

	// The operator-controlled environment variables filtered from
	// the process environment, based on named HTTPBIN_ prefix.
	env map[string]string

	// Pre-computed error message for the /redirect-to endpoint, based on
	// -allowed-redirect-domains/ALLOWED_REDIRECT_DOMAINS
	forbiddenRedirectError string

	// The app's http handler
	handler http.Handler

	// Pre-rendered templates
	indexHTML     []byte
	formsPostHTML []byte

	// Pre-computed map of special cases for the /status endpoint
	statusSpecialCases map[int]*statusCase

	// Optional function to control which headers are excluded from the
	// /headers response
	excludeHeadersProcessor headersProcessorFunc

	// Max number of SSE events to send, based on rough estimate of single
	// event's size
	maxSSECount int64

	RequestConfig
}

// DefaultParams defines default parameter values
type DefaultParams struct {
	// for the /drip endpoint
	DripDuration time.Duration
	DripDelay    time.Duration
	DripNumBytes int64

	// for the /sse endpoint
	SSECount    int
	SSEDuration time.Duration
	SSEDelay    time.Duration
}

// Observer is a function that will be called with the details of a handled
// request, which can be used for logging, instrumentation, etc

type RequestConfig struct {
	// Max size of an incoming request or generated response body, in bytes
	MaxBodySize int64
	// Max duration of a request, for those requests that allow user control
	// over timing (e.g. /delay)
	MaxDuration time.Duration
	// Observer called with the result of each handled request

	// Default parameter values
	DefaultParams DefaultParams

	// Set of hosts to which the /redirect-to endpoint will allow redirects
	AllowedRedirectDomains map[string]struct{}
}

func New(ctx *ctx.Ctx) *HttpBin {
	return &HttpBin{
		ctx:      ctx,
		logger:   ctx.Logger(),
		hostname: DefaultHostname,
		RequestConfig: RequestConfig{
			MaxBodySize: DefaultMaxBodySize,
			MaxDuration: DefaultMaxDuration,
		},
	}
}

func (h *HttpBin) StartServer() {
	h.g = gin.Default()
}

func (h *HttpBin) AddRouters() {
	g := h.g

	g.DELETE("/delete", h.RequestWithBody)
	g.GET("/{$}", h.Index)
	g.GET("/encoding/utf8", h.Index)
	g.GET("/forms/post", h.Index)
	g.GET("/get", h.Get)
	g.GET("/websocket/echo", h.WebsocketEcho)
	g.HEAD("/head", h.Get)
	g.PATCH("/patch", h.RequestWithBody)
	g.POST("/post", h.RequestWithBody)
	g.PUT("/put", h.RequestWithBody)

	g.Any("/absolute-redirect/{numRedirects}", h.AbsoluteRedirect)
	g.Any("/anything", h.AnyThing)
	g.Any("/anything/", h.AnyThing)
	g.Any("/base64/{data}", h.Base64)
	g.Any("/base64/{operation}/{data}", h.Base64)
	g.Any("/basic-auth/{user}/{passwd}", h.BasicAuth)
	g.Any("/bearer", h.Bearer)
	g.Any("/bytes/{numBytes}", h.Bytes)
	g.Any("/cache", h.Cache)
	g.Any("/cache/{numSeconds}", h.CacheControl)
	g.Any("/cookies", h.Cookies)
	g.Any("/cookies/delete", h.DeleteCookies)
	g.Any("/cookies/set", h.SetCookies)
	g.Any("/deflate", h.Deflate)
	g.Any("/delay/{duration}", h.Delay)

}
