package ctx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	// 1. 创建一个通道用来接收信号
	sigChan := make(chan os.Signal, 1)
	// 2. 监听指定信号
	// syscall.SIGINT 对应 Ctrl+C
	// syscall.SIGTERM 对应 kill 命令发送的终止信号（建议也加上）
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// 3. 启动一个 goroutine 监听信号
	go func() {
		var sig os.Signal
		select {
		case sig = <-sigChan:
			log.Info("received signal", zap.String("signal", sig.String()))
			fmt.Printf("exit by sig: %s\n", sig.String())
			// 4. 调用你的退出函数
			cancel()
			return
		}
	}()

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
