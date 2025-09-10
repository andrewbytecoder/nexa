// Package cmdregister 提供命令注册功能，用于统一管理所有子命令
package cmdregister

import (
	"sync"

	"github.com/nexa/pkg/cmd/gops"
	"github.com/nexa/pkg/cmd/httpstat"
	"github.com/nexa/pkg/cmd/psutil"
	"github.com/nexa/pkg/cmd/version"
	"github.com/nexa/pkg/ctx"
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
	n.cmdList = append(n.cmdList, cmd...)
	return n
}

// RegisterCmd 注册所有子命令到主命令中
func (n *NexaCommand) RegisterCmd(ctx *ctx.Ctx) {
	n.AddCommand(httpstat.GetHttpCmd(ctx))
	n.AddCommand(psutil.GetPsUtilCmd(ctx))
	n.AddCommand(gops.GetGoPsCmd(ctx))

	n.AddCommand(version.GetVersionCmd(ctx))
	for _, v := range n.cmdList {
		n.cmd.AddCommand(v)
	}
}

// Execute 执行命令行程序
func Execute() {
	GetNexaCommand().RegisterCmd(ctx.New())
	if err := GetNexaCommand().cmd.Execute(); err != nil {
		panic(err)
	}
}

// AddCommand 全局函数，用于向单例中添加命令
func AddCommand(cmd []*cobra.Command) {
	GetNexaCommand().AddCommand(cmd)
}
