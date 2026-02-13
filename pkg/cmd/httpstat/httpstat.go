package httpstat

import (
	"github.com/nexa/pkg/ctx"
	httpstat "github.com/nexa/pkg/net/httpstat"
	"github.com/spf13/cobra"
)

func GetHttpCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmdHttpStat(ctx))

	return cmds
}

// newCmdHttpStat returns a cobra command for fetching versions
func newCmdHttpStat(ctx *ctx.Ctx) *cobra.Command {
	httpStat := httpstat.NewHttpStat(ctx)

	cmd := &cobra.Command{
		Use:     "httpstat",
		Short:   "nexa httpstat url",
		Long:    `nexa httpstat url -X GET.`,
		Example: `nexa httpstat www.google.com -X GET -H "Accept: application/json, text/plain, */*"`,
		// stop printing usage when the command errors
		SilenceUsage: true,
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		httpStat.RunHttpStat(args[0])
	}

	httpStat.ParseFlags(cmd)
	return cmd
}

// newCmdHttpStat returns a cobra command for fetching versions
func newCmdHttpBinStat(ctx *ctx.Ctx) *cobra.Command {
	httpStat := httpstat.NewHttpStat(ctx)

	cmd := &cobra.Command{
		Use:     "httpstat",
		Short:   "nexa httpstat url",
		Long:    `nexa httpstat url -X GET.`,
		Example: `nexa httpstat www.google.com -X GET -H "Accept: application/json, text/plain, */*"`,
		// stop printing usage when the command errors
		SilenceUsage: true,
	}
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		httpStat.RunHttpStat(args[0])
	}

	httpStat.ParseFlags(cmd)
	return cmd
}
