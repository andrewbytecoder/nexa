package cmd

import (
	"github.com/nexa/pkg/cmd/version"
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
	return instance
}

func (n *NexaCommand) AddCommand(cmd []*cobra.Command) {
	n.cmdList = append(n.cmdList, cmd...)
}

func Execute() {
	rootCmd := GetNexaCommand()
	cobra.CheckErr(rootCmd.cmd.Execute())
}

func cmdRegiste() {
	rootCmd := GetNexaCommand()
	rootCmd.AddCommand(version.GetVersionCmd())
	rootCmd.AddCommand(NewCmdHttpstat())
}
