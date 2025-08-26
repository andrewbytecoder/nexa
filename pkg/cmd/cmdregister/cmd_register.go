package cmdregister

import (
	"github.com/nexa/pkg/cmd/httpstat"
	"github.com/nexa/pkg/cmd/tcpterm"
	"github.com/nexa/pkg/cmd/version"
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"sync"
)

type NexaCommand struct {
	cmd     *cobra.Command
	cmdList []*cobra.Command
}

var (
	instance *NexaCommand
	once     sync.Once
)

func NewNexaCommand() *NexaCommand {
	cmd := &cobra.Command{
		Use:   "nexa",
		Short: "Nexa is a command line tool for managing your nexa",
		Long:  `Nexa is a command line tool for managing your nexa`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	return &NexaCommand{
		cmd:     cmd,
		cmdList: []*cobra.Command{},
	}
}

func GetNexaCommand() *NexaCommand {
	once.Do(func() {
		cmd := NewNexaCommand()
		instance = cmd
	})

	instance.cmd.Flags().SetInterspersed(false)

	return instance
}

func (n *NexaCommand) AddCommand(cmd []*cobra.Command) *NexaCommand {
	n.cmdList = append(n.cmdList, cmd...)
	return n
}

func (n *NexaCommand) RegisterCmd(ctx *ctx.Ctx) {
	n.AddCommand(httpstat.GetHttpCmd(ctx))
	n.AddCommand(tcpterm.GetTcpTermCmd(ctx))
	n.AddCommand(version.GetVersionCmd(ctx))

	for _, v := range n.cmdList {
		n.cmd.AddCommand(v)
	}
}

func Execute() {
	GetNexaCommand().RegisterCmd(ctx.New())
	if err := GetNexaCommand().cmd.Execute(); err != nil {
		panic(err)
	}
}

func AddCommand(cmd []*cobra.Command) {
	GetNexaCommand().AddCommand(cmd)
}
