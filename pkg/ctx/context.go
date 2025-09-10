package ctx

import (
	"context"

	"github.com/nexa/pkg/utils"
	"go.uber.org/zap"
)

type Ctx struct {
	ctx    context.Context
	log    *zap.Logger
	cancel context.CancelFunc
}

func New() *Ctx {
	log, err := utils.NewLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Ctx{
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Ctx) Logger() *zap.Logger {
	return c.log
}

func (c *Ctx) Context() context.Context {
	return c.ctx
}
