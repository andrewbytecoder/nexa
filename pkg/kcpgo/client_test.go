package kcpgo

import (
	"os"
	"strconv"
	"testing"
	"time"

	ctx2 "github.com/nexa/pkg/ctx"
)

func TestKcpClient(t *testing.T) {
	addr := os.Getenv("NEXA_KCP_TEST_ADDR")
	if addr == "" {
		t.Skip("set NEXA_KCP_TEST_ADDR to run integration test")
	}
	portStr := os.Getenv("NEXA_KCP_TEST_PORT")
	if portStr == "" {
		portStr = "12345"
	}

	ctx := ctx2.New()
	kcp := New(ctx)

	kcp.Addr = addr
	kcp.Port = mustAtoiPort(t, portStr)

	kcp.RequestInterval = 1000

	go func() {
		defer ctx.Cancel()
		time.Sleep(3 * time.Second)
	}()

	kcp.StartClient()

}

func mustAtoiPort(t *testing.T, s string) int {
	t.Helper()
	p, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("invalid port %q: %v", s, err)
	}
	return p
}
