package udp

import (
	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/kcpgo"
	"github.com/spf13/cobra"
)

func GetCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newKcpClient(ctx))
	cmds = append(cmds, newKcpServer(ctx))

	return cmds
}

// newClient returns a cobra command for fetching versions
func newKcpClient(ctx *ctx.Ctx) *cobra.Command {
	kcp := kcpgo.New(ctx)

	cmd := &cobra.Command{
		Use:     "kcpclient",
		Short:   "nexa kcpclient ",
		Long:    `nexa kcpclient --addr=127.0.0.1 --port=12345 --interval=100 --body="request data"`,
		Example: `nexa kcpclient --addr=127.0.0.1 --port=12345 --interval=100 --body="request data"`,
		// stop printing usage when the command errors
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			kcp.StartClient()
		},
	}

	kcp.ParseFlags(cmd)

	return cmd
}

// newServer returns a cobra command for fetching versions
func newKcpServer(ctx *ctx.Ctx) *cobra.Command {
	kcp := kcpgo.New(ctx)

	cmd := &cobra.Command{
		Use:     "kcpserver",
		Short:   "nexa kcpserver ",
		Long:    `nexa kcpserver --addr=127.0.0.1 --port=12345`,
		Example: `nexa kcpserver --addr=127.0.0.1 --port=12345`,
		// stop printing usage when the command errors
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			kcp.StartServer()
		},
	}

	kcp.ParseFlags(cmd)

	return cmd
}
