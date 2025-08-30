package psutil

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
)

type PsHost struct {
	psUtil *PsUtil

	// Command line flags.
	// Command line flags.
	readable      bool
	showType      string
	usagePath     string
	allPartitions bool
}

func NewPsHost(psUtil *PsUtil) *PsHost {
	return &PsHost{
		psUtil: psUtil,
	}
}

func (psHost *PsHost) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psHost.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psHost.showType, "type", "t", "all", strings.Join([]string{tAll, tInfo, tUserStat}, "|"))
	cmd.Flags().StringVarP(&psHost.usagePath, "path", "u", "", "if not set depend on -a")
	cmd.Flags().BoolVarP(&psHost.allPartitions, "all", "a", true, "all partitions")
}

func (psHost *PsHost) GetHostInfo() {
	if psHost.showType == tAll || psHost.showType == tInfo {
		psHost.showInfo()
	}

	if psHost.showType == tAll || psHost.showType == tUserStat {
		psHost.showUseStat()
	}
}

func (psHost *PsHost) showInfo() {
	info, err := host.Info()
	if err != nil {
		psHost.psUtil.logger.Error("unable to get host info", zap.Error(err))
		return
	}

	psHost.printHostInfoTable(info)
}

func (psHost *PsHost) showUseStat() {
	// 获取使用状态
	userStats, err := host.Users()
	if err != nil {
		psHost.psUtil.logger.Error("unable to get host use stat", zap.Error(err))
		return
	}
	psHost.printUserStats(userStats)
}

func (psHost *PsHost) printUserStats(usersStats []host.UserStat) {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"User", "Terminal", "Host", "Started"})
	// 将 Unix 时间戳转换为可读时间
	for _, u := range usersStats {
		startedTime := time.Unix(int64(u.Started), 0).Format("2006-01-02 15:04:05")
		row := []string{
			u.User,
			u.Terminal,
			u.Host,
			startedTime,
		}
		err := table.Append(row)
		if err != nil {
			psHost.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err := table.Render()
	if err != nil {
		psHost.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

func (psHost *PsHost) printHostInfoTable(info *host.InfoStat) {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Field", "Value"})
	// 辅助函数：将秒转换为易读的天/小时格式
	formatUptime := func(seconds uint64) string {
		days := seconds / 86400
		hours := (seconds % 86400) / 3600
		mins := (seconds % 3600) / 60
		if days > 0 {
			return fmt.Sprintf("%d (%d days, %d hours)", seconds, days, hours)
		}
		return fmt.Sprintf("%d (%d:%02d)", seconds, hours, mins)
	}

	// 辅助函数：将 Unix 时间戳转为可读时间
	formatBootTime := func(timestamp uint64) string {
		t := time.Unix(int64(timestamp), 0)
		return fmt.Sprintf("%d (%s)", timestamp, t.Format("2006-01-02 15:04:05"))
	}

	// 添加数据行
	data := [][]string{
		{"Hostname", info.Hostname},
		{"Uptime", formatUptime(info.Uptime)},
		{"Boot Time", formatBootTime(info.BootTime)},
		{"Processes", fmt.Sprintf("%d", info.Procs)},
		{"OS", info.OS},
		{"Platform", info.Platform},
		{"Platform Family", info.PlatformFamily},
		{"Platform Version", info.PlatformVersion},
		{"Kernel Version", info.KernelVersion},
		{"Kernel Arch", info.KernelArch},
		{"Virtualization", info.VirtualizationSystem},
		{"Role", info.VirtualizationRole},
		{"Host ID", info.HostID},
	}

	for _, v := range data {
		err := table.Append(v)
		if err != nil {
			psHost.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err := table.Render()
	if err != nil {
		psHost.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}
