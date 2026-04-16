package httpbin

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/gops/agent"
	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/utils"
	"github.com/spf13/cobra"
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

// DefaultDefaultParams defines the DefaultParams that are used by default. In
// general, these should match the original httpbin.org's defaults.
var DefaultDefaultParams = DefaultParams{
	DripDuration: 2 * time.Second,
	DripDelay:    2 * time.Second,
	DripNumBytes: 10,
	SSECount:     10,
	SSEDuration:  5 * time.Second,
	SSEDelay:     0,
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
	// http port
	Port int
	// https port
	HttpsPort int

	certPath string
	certFile string
	keyFile  string

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
			MaxBodySize:   DefaultMaxBodySize,
			MaxDuration:   DefaultMaxDuration,
			DefaultParams: DefaultDefaultParams,
		},
	}
}

func (httpBin *HttpBin) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&httpBin.Port, "http-port", "p", 8080, "HTTP port")
	cmd.Flags().IntVarP(&httpBin.HttpsPort, "https-port", "s", 8443, "HTTPS port")
	cmd.Flags().StringVarP(&httpBin.certPath, "cert-path", "c", "", "cert path")
}

func (h *HttpBin) StartServer() {
	// 创建路由引擎
	h.g = gin.Default()
	// 添加路由
	h.AddRouters()

	if err := agent.Listen(agent.Options{}); err != nil {
		h.logger.Error("Failed to start gops agent", zap.Error(err))
		return
	}

	go func() {
		//	 启动http服务
		h.logger.Info("httpbin is ready to serve requests", zap.Int("port", h.Port))
		err := h.Run(fmt.Sprintf(":%d", h.Port))
		if err != nil {
			h.logger.Error("Failed to start http server", zap.Error(err))
		}
	}()
	// 启动https服务
	h.logger.Info("httpbin is ready to serve requests", zap.Int("port", h.HttpsPort))

	certfile, keyfile := h.GetCertFiles()
	err := h.RunTLS(fmt.Sprintf(":%d", h.HttpsPort), certfile, keyfile)
	if err != nil {
		h.logger.Error("Failed to start https server", zap.Error(err))
		return
	}
}

func (h *HttpBin) Run(addr string) error {
	return h.g.Run(addr)
}

func (h *HttpBin) RunTLS(addr string, certFile string, keyFile string) error {
	return h.g.RunTLS(addr, certFile, keyFile)
}

func (h *HttpBin) GetCertFiles() (string, string) {
	var certFile, keyFile string

	certPath := utils.CleanPathSuffix(h.certPath, "/")

	if certPath == "" {
		if runtime.GOOS == "windows" {
			certFile = "server.crt"
			keyFile = "server.key"
		} else {
			certFile = "/home/nexa/certs/server.crt"
			keyFile = "/home/nexa/certs/server.key"
		}
	} else {
		certFile = certPath + "/server.crt"
		keyFile = certPath + "/server.key"
	}

	return certFile, keyFile
}

func (h *HttpBin) AddRouters() {
	g := h.g

	g.DELETE("/delete", h.RequestWithBody)
	g.GET("/", h.Index)
	g.GET("/encoding/utf8", h.EncodingUTF8)
	g.GET("/forms/post", h.FormsPost)
	g.GET("/get", h.Get)
	g.GET("/websocket/echo", h.WebsocketEcho)
	g.HEAD("/head", h.Get)
	g.PATCH("/patch", h.RequestWithBody)
	g.POST("/post", h.RequestWithBody)
	g.PUT("/put", h.RequestWithBody)

	g.Any("/absolute-redirect/:numRedirects", h.AbsoluteRedirect)
	g.Any("/anything", h.AnyThing)
	g.Any("/anything/", h.AnyThing)
	// Use a single catch-all route to avoid wildcard conflicts like:
	// /base64/:data  vs  /base64/:operation/:data
	g.Any("/base64/*path", h.Base64)
	g.Any("/basic-auth/:user/:password", h.BasicAuth)
	g.Any("/bearer", h.Bearer)
	g.Any("/bytes/:numBytes", h.Bytes)
	g.Any("/cache", h.Cache)
	g.Any("/cache/:numSeconds", h.CacheControl)
	g.Any("/cookies", h.Cookies)
	g.Any("/cookies/delete", h.DeleteCookies)
	g.Any("/cookies/set", h.SetCookies)
	g.Any("/deflate", h.Deflate)
	g.Any("/delay/:duration", h.Delay)
	g.Any("/deny", h.Deny)
	g.Any("/digest-auth/:qop/:user/:password", h.DigestAuth)
	g.Any("/digest-auth/:qop/:user/:password/:algorithm", h.DigestAuth)
	g.Any("/drip", h.Drip)
	g.Any("/dump/request", h.DumpRequest)
	g.Any("/env", h.Env)
	g.Any("/etag/:etag", h.ETag)
	g.Any("/gzip", h.Gzip)
	g.Any("/headers", h.Headers)
	g.Any("/hidden-basic-auth/:user/:password", h.HiddenBasicAuth)
	g.Any("/hostname", h.HostName)
	g.Any("/html", h.Html)
	g.Any("/image", h.ImageAccept)
	g.Any("/image/:kind", h.Image)
	g.Any("/ip", h.IP)
	g.Any("/json", h.JSON)
	g.Any("/links/:numLinks", h.Links)
	g.Any("/links/:numLinks/:offset", h.Links)
	g.Any("/range/:numBytes", h.Range)
	g.Any("/redirect-to", h.RedirectTo)
	g.Any("/redirect/:numRedirects", h.Redirect)
	g.Any("/relative-redirect/:numRedirects", h.RelativeRedirect)
	g.Any("/response-headers", h.ResponseHeaders)
	g.Any("/robots.txt", h.Robots)
	g.Any("/sse", h.SSE)
	g.Any("/status/:status", h.Status)
	g.Any("/stream-bytes/:numBytes", h.StreamBytes)
	g.Any("/stream/:numLines", h.Stream)
	g.Any("/trailers", h.Trailers)
	g.Any("/unstable", h.Unstable)
	g.POST("/upload", h.RequestWithBodyDiscard)
	g.PUT("/upload", h.RequestWithBodyDiscard)
	g.PATCH("/upload", h.RequestWithBodyDiscard)
	g.Any("/user-agent", h.UserAgent)
	g.Any("/uuid", h.UUID)
	g.Any("/xml", h.XML)

	// existing httpbin endpoints that we do not support
	g.Any("/brotli", notImplementedHandler)
}
