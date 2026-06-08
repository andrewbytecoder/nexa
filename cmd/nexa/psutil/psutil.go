package psutil

import (
	"github.com/nexa/cmd/nexa/complete"
	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/psutil"
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
		Use:               "psutil",
		Short:             "psutil ps for human",
		Long:              `nexa psutil [command].`,
		Example:           `nexa psutil memory"`,
		ValidArgsFunction: completePsutilSubcommands,
		// stop printing usage when the command errors
		SilenceUsage: true,
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
	memCmd.RegisterFlagCompletionFunc("type", completeMemType)
	memCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)

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
	cpuCmd.RegisterFlagCompletionFunc("type", completeCpuType)
	cpuCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)
	cpuCmd.RegisterFlagCompletionFunc("per-cpu", complete.Bool)

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
	diskCmd.RegisterFlagCompletionFunc("type", completeDiskType)
	diskCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)
	diskCmd.RegisterFlagCompletionFunc("all", complete.Bool)
	diskCmd.RegisterFlagCompletionFunc("path", complete.Dir)

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
	hostCmd.RegisterFlagCompletionFunc("type", completeHostType)
	hostCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)

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
	loadCmd.RegisterFlagCompletionFunc("type", completeLoadType)
	loadCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)

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
	netCmd.RegisterFlagCompletionFunc("type", completeNetType)
	netCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)
	netCmd.RegisterFlagCompletionFunc("kind", completeNetKind)
	netCmd.RegisterFlagCompletionFunc("pernic", complete.Bool)
	netCmd.RegisterFlagCompletionFunc("percpu", complete.Bool)

	// 进程信息统计
	processCmd := &cobra.Command{
		Use:     "process",
		Short:   "psutil process",
		Long:    `psutil process [global options] command [command options] [arguments...].`,
		Example: `nexa psutil process"`,
		Run: func(cmd *cobra.Command, args []string) {
			psUtil.GetProcessHandler().GetProcessInfo()
		},
	}
	cmd.AddCommand(processCmd)
	psUtil.GetProcessHandler().ParseFlags(processCmd)
	processCmd.RegisterFlagCompletionFunc("type", completeProcessType)
	processCmd.RegisterFlagCompletionFunc("human-readable", complete.Bool)
}

// --- 自动补全辅助函数 ---

// completePsutilSubcommands 为 psutil 父命令补全子命令名称
func completePsutilSubcommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"memory", "cpu", "disk", "host", "load", "net", "process"}, toComplete)
}

// --- 各子命令 --type 参数补全函数 ---

func completeMemType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "mem", "swap", "swapDev"}, toComplete)
}

func completeCpuType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "times", "percent", "info"}, toComplete)
}

func completeDiskType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "usage", "IOCounter"}, toComplete)
}

func completeHostType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "info", "userStat"}, toComplete)
}

func completeLoadType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "loadAvg", "loadMisc"}, toComplete)
}

func completeNetType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "netIfConfig", "netIOCounter", "netConnections", "netConntrack", "netPids"}, toComplete)
}

func completeNetKind(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "unix", "inet", "inet4", "inet6"}, toComplete)
}

func completeProcessType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.Values([]string{"all", "netProcess", "netProcessByPid"}, toComplete)
}
