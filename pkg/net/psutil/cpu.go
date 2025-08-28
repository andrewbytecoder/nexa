package psutil

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strings"
	"text/tabwriter"
)

type PsCpu struct {
	psUtil *PsUtil

	// Command line flags.
	// Command line flags.
	readable bool
	showType string
	perCpu   bool
}

func NewPsCpu(psUtil *PsUtil) *PsCpu {
	return &PsCpu{
		psUtil: psUtil,
	}
}

const (
	tTimes = "times"
	tInfo  = "info"
)

func (psCpu *PsCpu) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psCpu.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psCpu.showType, "type", "t", "all", strings.Join([]string{tAll, tTimes, tInfo}, "|"))
	cmd.Flags().BoolVarP(&psCpu.perCpu, "per-cpu", "p", true, "default true")

}

func (psCpu *PsCpu) GetCpuInfo() {
	if psCpu.showType == tAll || psCpu.showType == tTimes {
		psCpu.ShowCpuTimes()
	}

	if psCpu.showType == tAll || psCpu.showType == tInfo {
		psCpu.ShowCpuInfo()
	}
}

func (psCpu *PsCpu) ShowCpuTimes() {
	times, err := cpu.Times(psCpu.perCpu)
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu times", zap.Error(err))
		return
	}

	PrintCPUSummary(times)
}

func (psCpu *PsCpu) ShowCpuInfo() {
	times, err := cpu.Info()
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu info", zap.Error(err))
		return
	}

	PrintCPUInfoTable(times)
}

// PrintCPUSummary 将 []TimesStat 格式化输出为表格
func PrintCPUSummary(stats []cpu.TimesStat) {
	w := tabwriter.NewWriter(os.Stdout, 10, 4, 3, ' ', 0)

	// 表头
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"CPU", "User", "System", "Idle", "Iowait", "Irq", "Softirq", "Steal", "Guest", "GuestNice", "Nice")

	// 分隔线（可选）
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"-----", "------", "--------", "--------", "--------", "---", "---------", "-------", "-------", "-----------", "----")

	// 数据行
	for _, s := range stats {
		_, _ = fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\n",
			s.CPU,
			s.User,
			s.System,
			s.Idle,
			s.Iowait,
			s.Irq,
			s.Softirq,
			s.Steal,
			s.Guest,
			s.GuestNice,
			s.Nice,
		)
	}

	// 刷新输出
	_ = w.Flush()
}

// PrintCPUInfoTable 美观输出 CPU 信息表格
func PrintCPUInfoTable(info []cpu.InfoStat) {
	// 使用空格作为分隔符，配合固定宽度控制对齐
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)

	// 自定义列宽格式（可根据实际数据调整）
	const (
		colCPU        = "%-2d"
		colVendorID   = "%-12s"
		colFamily     = "%-6s"
		colModel      = "%-5s"
		colStepping   = "%-9d"
		colPhysicalID = "%-11s"
		colCoreID     = "%-8s"
		colCores      = "%-8d"
		colModelName  = "%-38s"
		colMhz        = "%-10.2f"
		colCacheSize  = "%-10d"
		colFlags      = "%-22s"
		colMicrocode  = "%-10s"
	)

	// 表头
	fmt.Fprintf(w, colCPU+"\t"+colVendorID+"\t"+colFamily+"\t"+colModel+"\t"+
		colStepping+"\t"+colPhysicalID+"\t"+colCoreID+"\t"+colCores+"\t"+
		colModelName+"\t"+colMhz+"\t"+colCacheSize+"\t"+colFlags+"\t"+colMicrocode+"\n",
		"CPU", "VendorID", "Family", "Model", "Stepping", "PhysicalID",
		"CoreID", "Cores", "ModelName", "Mhz", "CacheSize", "Flags", "Microcode")

	// 分隔线（对齐表头）
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"----", "------------", "------", "-----", "---------", "-----------",
		"--------", "--------", "--------------------------------------", "----------",
		"----------", "----------------------", "----------")

	// 数据行
	for _, cpu := range info {
		flags := strings.Join(cpu.Flags, ",")
		if len(flags) > 22 {
			flags = flags[:19] + "..."
		}

		fmt.Fprintf(w,
			colCPU+"\t"+colVendorID+"\t"+colFamily+"\t"+colModel+"\t"+
				colStepping+"\t"+colPhysicalID+"\t"+colCoreID+"\t"+colCores+"\t"+
				colModelName+"\t"+colMhz+"\t"+colCacheSize+"\t"+colFlags+"\t"+colMicrocode+"\n",
			cpu.CPU,
			cpu.VendorID,
			cpu.Family,
			cpu.Model,
			cpu.Stepping,
			cpu.PhysicalID,
			cpu.CoreID,
			cpu.Cores,
			cpu.ModelName,
			cpu.Mhz,
			cpu.CacheSize,
			flags,
			cpu.Microcode,
		)
	}

	w.Flush()
}
