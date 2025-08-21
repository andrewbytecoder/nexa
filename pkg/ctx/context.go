package ctx

import (
	"context"
	"github.com/nexa/pkg/utils"
	"go.uber.org/zap"
)

type Ctx struct {
	ctx context.Context
	log *zap.Logger
}

func New() *Ctx {
	log, err := utils.NewLogger()
	if err != nil {
		panic(err)
	}
	return &Ctx{log: log}
}

func (c *Ctx) Logger() *zap.Logger {
	return c.log
}

func (c *Ctx) Context() context.Context {
	return c.ctx
}
