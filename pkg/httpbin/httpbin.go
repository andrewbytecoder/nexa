package httpbin

import (
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

type HttpBin struct {
	ctx    *ctx.Ctx
	logger *zap.Logger
	g      *gin.Engine

	// The hostname to expose via /hostname.
	hostname string

	// Optional prefix under which the app will be served
	prefix string

	RequestConfig
}

type RequestConfig struct {
	// Max size of an incoming request or generated response body, in bytes
	MaxBodySize int64
	// Max duration of a request, for those requests that allow user control
	// over timing (e.g. /delay)
	MaxDuration time.Duration
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

	g.Any("/absolute-redirect/{numRedirects}")

}
