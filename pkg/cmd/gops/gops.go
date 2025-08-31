package gops

import (
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
	psUtil := gops.NewGoPs(ctx)

	cmd := &cobra.Command{
		Use:     "gops",
		Short:   "gops ps for human",
		Long:    `nexa gops [command].`,
		Example: `nexa gops memory"`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
		},
	}

	psUtil.ParseFlags(cmd)

	addSubCmd(psUtil, cmd)

	return cmd
}

func addSubCmd(psUtil *gops.PsUtil, cmd *cobra.Command) {
	// 内存使用情况统计
	memCmd := &cobra.Command{
		Use:     "memory",
		Short:   "gops memory",
		Long:    `gops memory [global options] command [command options] [arguments...].`,
		Example: `nexa gops memory -H=true"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetMemoryHandler().GetMemInfo()
		},
	}
	cmd.AddCommand(memCmd)

	psUtil.GetMemoryHandler().ParseFlags(memCmd)
	// cpu使用情况统计
	cpuCmd := &cobra.Command{
		Use:     "cpu",
		Short:   "gops cpu",
		Long:    `gops cpu [global options] command [command options] [arguments...].`,
		Example: `nexa gops cpu -H=true"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetCpuHandler().GetCpuInfo()
		},
	}
	cmd.AddCommand(cpuCmd)
	psUtil.GetCpuHandler().ParseFlags(cpuCmd)

	// 磁盘使用情况统计
	diskCmd := &cobra.Command{
		Use:     "disk",
		Short:   "gops disk",
		Long:    `gops disk [global options] command [command options] [arguments...].`,
		Example: `nexa gops disk"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetDiskHandler().GetDiskInfo()
		},
	}
	cmd.AddCommand(diskCmd)
	psUtil.GetDiskHandler().ParseFlags(diskCmd)

	// 主机信息统计
	hostCmd := &cobra.Command{
		Use:     "host",
		Short:   "gops host",
		Long:    `gops host [global options] command [command options] [arguments...].`,
		Example: `nexa gops host"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetHostHandler().GetHostInfo()
		},
	}
	cmd.AddCommand(hostCmd)
	psUtil.GetHostHandler().ParseFlags(hostCmd)

	// 负载信息统计
	loadCmd := &cobra.Command{
		Use:     "load",
		Short:   "gops load",
		Long:    `gops load [global options] command [command options] [arguments...].`,
		Example: `nexa gops load"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetLoadHandler().GetLoadInfo()
		},
	}
	cmd.AddCommand(loadCmd)
	psUtil.GetLoadHandler().ParseFlags(loadCmd)

	// 网络信息统计
	netCmd := &cobra.Command{
		Use:     "net",
		Short:   "gops net",
		Long:    `gops net [global options] command [command options] [arguments...].`,
		Example: `nexa gops net"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetNetHandler().GetnetInfo()
		},
	}
	cmd.AddCommand(netCmd)
	psUtil.GetNetHandler().ParseFlags(netCmd)

	// 进程信息统计
	processCmd := &cobra.Command{
		Use:     "process",
		Short:   "gops process",
		Long:    `gops process [global options] command [command options] [arguments...].`,
		Example: `nexa gops process"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetProcessHandler().GetProcessInfo()
		},
	}
	cmd.AddCommand(processCmd)
	psUtil.GetProcessHandler().ParseFlags(processCmd)
}
