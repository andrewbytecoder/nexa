package kcpgo

import (
	"crypto/sha1"
	"fmt"

	"github.com/google/gops/agent"
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"github.com/xtaci/kcp-go/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
)

type Kcp struct {
	ctx *ctx.Ctx
	log *zap.Logger

	Port            int    // udp使用的端口
	Addr            string // udp监听地址
	RequestInterval int    // 请求间隔，单位毫秒
	SendData        string // 发送数据
}

func New(ctx *ctx.Ctx) *Kcp {
	return &Kcp{
		ctx: ctx,
		log: ctx.Logger(),
	}
}

func (k *Kcp) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&k.Port, "port", "p", 12345, "udp port")
	cmd.Flags().StringVarP(&k.Addr, "addr", "a", "0.0.0.0", "udp addr")
	cmd.Flags().IntVarP(&k.RequestInterval, "interval", "i", 1000, "request interval")
	cmd.Flags().StringVarP(&k.SendData, "body", "b", "", "send data")

}

func (k *Kcp) StartServer() {

	if err := agent.Listen(agent.Options{}); err != nil {
		k.log.Error("Failed to start gops agent", zap.Error(err))
		return
	}

	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	requestAddr := fmt.Sprintf("%s:%d", k.Addr, k.Port)
	k.log.Info("connecting to ", zap.String("addr", requestAddr))
	listener, err := kcp.ListenWithOptions(requestAddr, block, 10, 3)
	if err != nil {
		k.log.Info("listen error", zap.Error(err))
		return
	}
	defer listener.Close()

	// spin-up the client
	for {
		s, err := listener.AcceptKCP()
		if err != nil {
			k.log.Error("accept error", zap.Error(err))
			return
		}

		go k.handleEcho(s)
	}
}

// handleEcho send back everything it received
func (k *Kcp) handleEcho(conn *kcp.UDPSession) {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			k.log.Error("read error", zap.Error(err))
			return
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			k.log.Error("write error", zap.Error(err))
			return
		}
	}
}
