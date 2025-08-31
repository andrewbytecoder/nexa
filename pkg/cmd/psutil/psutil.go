package psutil

import (
	"github.com/nexa/pkg/ctx"
	psutil "github.com/nexa/pkg/net/psutil"
	"github.com/spf13/cobra"
)

func GetPsUtilCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmd(ctx))

	return cmds
}

// newCmdTcpTerm returns a cobra command for fetching versions
func newCmd(ctx *ctx.Ctx) *cobra.Command {
	psUtil := psutil.NewPsutil(ctx)

	cmd := &cobra.Command{
		Use:     "psutil",
		Short:   "psutil ps for human",
		Long:    `nexa psutil [command].`,
		Example: `nexa psutil memory"`,
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

func addSubCmd(psUtil *psutil.PsUtil, cmd *cobra.Command) {
	// 内存使用情况统计
	memCmd := &cobra.Command{
		Use:     "memory",
		Short:   "psutil memory",
		Long:    `psutil memory [global options] command [command options] [arguments...].`,
		Example: `nexa psutil memory -H=true"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetMemoryHandler().GetMemInfo()
		},
	}
	cmd.AddCommand(memCmd)

	psUtil.GetMemoryHandler().ParseFlags(memCmd)
	// cpu使用情况统计
	cpuCmd := &cobra.Command{
		Use:     "cpu",
		Short:   "psutil cpu",
		Long:    `psutil cpu [global options] command [command options] [arguments...].`,
		Example: `nexa psutil cpu -H=true"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetCpuHandler().GetCpuInfo()
		},
	}
	cmd.AddCommand(cpuCmd)
	psUtil.GetCpuHandler().ParseFlags(cpuCmd)

	// 磁盘使用情况统计
	diskCmd := &cobra.Command{
		Use:     "disk",
		Short:   "psutil disk",
		Long:    `psutil disk [global options] command [command options] [arguments...].`,
		Example: `nexa psutil disk"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetDiskHandler().GetDiskInfo()
		},
	}
	cmd.AddCommand(diskCmd)
	psUtil.GetDiskHandler().ParseFlags(diskCmd)

	// 主机信息统计
	hostCmd := &cobra.Command{
		Use:     "host",
		Short:   "psutil host",
		Long:    `psutil host [global options] command [command options] [arguments...].`,
		Example: `nexa psutil host"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetHostHandler().GetHostInfo()
		},
	}
	cmd.AddCommand(hostCmd)
	psUtil.GetHostHandler().ParseFlags(hostCmd)

	// 负载信息统计
	loadCmd := &cobra.Command{
		Use:     "load",
		Short:   "psutil load",
		Long:    `psutil load [global options] command [command options] [arguments...].`,
		Example: `nexa psutil load"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetLoadHandler().GetLoadInfo()
		},
	}
	cmd.AddCommand(loadCmd)
	psUtil.GetLoadHandler().ParseFlags(loadCmd)

	// 网络信息统计
	netCmd := &cobra.Command{
		Use:     "net",
		Short:   "psutil net",
		Long:    `psutil net [global options] command [command options] [arguments...].`,
		Example: `nexa psutil net"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetNetHandler().GetnetInfo()
		},
	}
	cmd.AddCommand(netCmd)
	psUtil.GetNetHandler().ParseFlags(netCmd)
}
