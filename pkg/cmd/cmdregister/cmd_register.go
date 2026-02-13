// Package cmdregister 提供命令注册功能，用于统一管理所有子命令
package cmdregister

import (
	"sync"

	"github.com/spf13/cobra"
)

// NexaCommand 结构体用于封装主命令及其所有子命令
type NexaCommand struct {
	cmd     *cobra.Command
	cmdList []*cobra.Command
}

// 全局变量，用于实现单例模式
var (
	instance *NexaCommand
	once     sync.Once
)

// NewNexaCommand 创建一个新的NexaCommand实例
func NewNexaCommand() *NexaCommand {
	cmd := &cobra.Command{
		Use:   "nexa",
		Short: "Nexa is a command line tool",
		Long:  `Nexa is a command line tool for managing your nexa`,
		// stop printing usage when the command errors
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	return &NexaCommand{
		cmd:     cmd,
		cmdList: []*cobra.Command{},
	}
}

// GetNexaCommand 获取NexaCommand单例实例
func GetNexaCommand() *NexaCommand {
	once.Do(func() {
		cmd := NewNexaCommand()
		instance = cmd
	})

	instance.cmd.Flags().SetInterspersed(false)

	return instance
}

// AddCommand 添加子命令到命令列表中
func (n *NexaCommand) AddCommand(cmd []*cobra.Command) *NexaCommand {
	n.cmd.AddCommand(cmd...)
	return n
}

// Execute 执行命令行程序
func (n *NexaCommand) Execute() error {
	if err := n.cmd.Execute(); err != nil {
		return err
	}
	return nil
}

// AddCommand 全局函数，用于向单例中添加命令
func AddCommand(cmd []*cobra.Command) {
	GetNexaCommand().AddCommand(cmd)
}
