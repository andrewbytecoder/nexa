package gops

import (
	"strconv"

	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/gops"
	"github.com/spf13/cobra"
)

func GetGoPsCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmd(ctx))

	return cmds
}

// newCmdTcpTerm returns a cobra command for fetching versions
func newCmd(ctx *ctx.Ctx) *cobra.Command {
	psCmd := gops.NewGoPs(ctx)

	cmd := &cobra.Command{
		Use:   "gops",
		Short: "gops is a tool to list and diagnose Go processes.",
		Long:  `nexa gops [command].`,
		Example: `nexa gops <cmd> <pid|addr> ...
		gops <pid> # displays process info
		gops help  # displays this help message`,
		// stop printing usage when the command errors
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			processes()

			// Legacy support for `gops <pid>` command.
			//
			// When the second argument is provided as int as opposed to a sub-command
			// (like proc, version, etc), gops command effectively shortcuts that
			// to `gops process <pid>`.
			if len(args) > 1 {
				// See second argument appears to be a pid rather than a subcommand
				_, err := strconv.Atoi(args[1])
				if err == nil {
					err := ProcessInfo(args[1:])
					if err != nil {
						return
					} // shift off the command name
					return
				}
			}
		},
	}

	psCmd.ParseFlags(cmd)

	addSubCmd(psCmd, cmd)

	return cmd
}

func addSubCmd(psUtil *gops.PsCmd, cmd *cobra.Command) {
	cmd.AddCommand(ProcessCommand())
	cmd.AddCommand(TreeCommand())
	cmd.AddCommand(AgentCommands()...)
}
