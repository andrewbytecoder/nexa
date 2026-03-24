package synscan

import (
	"github.com/nexa/pkg/ctx"
	"go.uber.org/zap"
)

type Scanner struct {
	ctx    *ctx.Ctx
	logger *zap.Logger
}

func New(ctx *ctx.Ctx, logger *zap.Logger) *Scanner {
	return &Scanner{ctx: ctx, logger: logger}
}
