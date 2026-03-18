package kcpgo

import (
	"testing"
	"time"

	ctx2 "github.com/nexa/pkg/ctx"
)

func TestKcpClient(t *testing.T) {
	ctx := ctx2.New()
	kcp := New(ctx)

	kcp.Addr = "10.168.8.110"
	kcp.Port = 12345

	kcp.RequestInterval = 1000

	go func() {
		defer ctx.Cancel()
		time.Sleep(10 * time.Second)
	}()

	kcp.StartClient()

}
