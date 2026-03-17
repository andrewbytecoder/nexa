package kcpgo

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/google/gops/agent"
	"github.com/xtaci/kcp-go/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
)

func (k *Kcp) StartClient() {

	if err := agent.Listen(agent.Options{}); err != nil {
		k.log.Error("Failed to start gops agent", zap.Error(err))
		return
	}

	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	// wait for server to become ready
	time.Sleep(time.Second)

	requestAddr := fmt.Sprintf("%s:%d", k.Addr, k.Port)
	k.log.Info("connecting to ", zap.String("addr", requestAddr))
	// dial to the echo server
	bClosed := false
	sess, err := kcp.DialWithOptions(requestAddr, block, 10, 3)
	if err != nil {
		k.log.Info("dial error", zap.Error(err))
		return
	}
	defer func() {
		if !bClosed {
			sess.Close()
		}
	}()

	ticker := time.NewTicker(time.Duration(k.RequestInterval) * time.Millisecond)

	var data string
	if k.SendData != "" {
		data = k.SendData
	}

	go func() {
		select {
		case <-k.ctx.Context().Done():
			k.log.Warn("client stopped")
			bClosed = true
			sess.Close()
			return
		}
	}()

	for {
		select {
		case <-ticker.C:
			if data == "" {
				data = time.Now().String()
			}
			buf := make([]byte, len(data))
			k.log.Info("sending", zap.String("data", data))

			_, err = sess.Write([]byte(data))
			if err != nil {
				k.log.Info("write error", zap.Error(err))
				return
			}

			// read back the data
			_, err = io.ReadFull(sess, buf)
			if err != nil {
				k.log.Info("read error", zap.Error(err))
				log.Fatal(err)
				return
			}
			k.log.Info("recv", zap.String("data", string(buf)))
		}
	}

}
