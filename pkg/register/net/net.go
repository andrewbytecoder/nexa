package net

import (
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
)

func Cmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmdHttpStat(ctx))
	cmds = append(cmds, newCmdHttpBinStat(ctx))

	return cmds
}
