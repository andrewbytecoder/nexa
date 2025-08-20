package context

import (
	"github.com/nexa/pkg/utils"
	"go.uber.org/zap"
)

type Context struct {
	log *zap.Logger
}

func New() *Context {
	log, err := utils.NewLogger()
	if err != nil {
		panic(err)
	}
	return &Context{log: log}
}
