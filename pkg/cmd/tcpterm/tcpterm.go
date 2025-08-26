package tcpterm

import (
	"github.com/nexa/pkg/ctx"
	tcpterml "github.com/nexa/pkg/net/tcpterm"
	"github.com/spf13/cobra"
)

func GetTcpTermCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmdTcpTerm(ctx))

	return cmds
}

// newCmdTcpTerm returns a cobra command for fetching versions
func newCmdTcpTerm(ctx *ctx.Ctx) *cobra.Command {
	tcpTerm := tcpterml.NewTcpTerm(ctx)

	cmd := &cobra.Command{
		Use:     "tcpterm",
		Short:   "tcpterm - tcpdump for human",
		Long:    `tcpterm [global options] command [command options] [arguments...].`,
		Example: `nexa tcpterm www.google.com -X GET -H "Accept: application/json, text/plain, */*"`,
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		tcpTerm.RunTcpTerm()
	}

	tcpTerm.ParseFlags(cmd)
	return cmd
}
