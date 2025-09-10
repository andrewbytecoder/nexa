package psutil

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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

func (psCpu *PsCpu) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psCpu.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psCpu.showType, "type", "t", "all", strings.Join([]string{tAll, tTimes, tPercent, tInfo}, "|"))
	cmd.Flags().BoolVarP(&psCpu.perCpu, "per-cpu", "p", true, "default true")

}

func (psCpu *PsCpu) GetCpuInfo() {
	if psCpu.showType == tAll || psCpu.showType == tTimes {
		psCpu.showCpuTimes()
	}

	if psCpu.showType == tAll || psCpu.showType == tInfo {
		psCpu.showCpuInfo()
	}

	if psCpu.showType == tAll || psCpu.showType == tPercent {
		psCpu.showCpuPercent()
	}
}

func (psCpu *PsCpu) showCpuTimes() {
	times, err := cpu.Times(psCpu.perCpu)
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu times", zap.Error(err))
		return
	}

	printCPUSummary(times)
}

func (psCpu *PsCpu) showCpuPercent() {
	percents, err := cpu.Percent(time.Second*1, psCpu.perCpu)
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu times", zap.Error(err))
		return
	}

	fmt.Println("cpu percent:", percents)
	fmt.Println("------------------------------------------")

	// 采样间隔 1 秒
	timesPercent, err := getCPUPercent(1 * time.Second)
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu times", zap.Error(err))
		os.Exit(1)
	}

	printCPUPercentTable(timesPercent)
}

func (psCpu *PsCpu) showCpuInfo() {
	times, err := cpu.Info()
	if err != nil {
		psCpu.psUtil.logger.Error("unable to get cpu info", zap.Error(err))
		return
	}

	printCPUInfoTable(times)
}

// printCPUSummary 将 []TimesStat 格式化输出为表格
func printCPUSummary(stats []cpu.TimesStat) {
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

// printCPUInfoTable 美观输出 CPU 信息表格
func printCPUInfoTable(info []cpu.InfoStat) {
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
	_, _ = fmt.Fprintf(w, colCPU+"\t"+colVendorID+"\t"+colFamily+"\t"+colModel+"\t"+
		colStepping+"\t"+colPhysicalID+"\t"+colCoreID+"\t"+colCores+"\t"+
		colModelName+"\t"+colMhz+"\t"+colCacheSize+"\t"+colFlags+"\t"+colMicrocode+"\n",
		"CPU", "VendorID", "Family", "Model", "Stepping", "PhysicalID",
		"CoreID", "Cores", "ModelName", "Mhz", "CacheSize", "Flags", "Microcode")

	// 分隔线（对齐表头）
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"----", "------------", "------", "-----", "---------", "-----------",
		"--------", "--------", "--------------------------------------", "----------",
		"----------", "----------------------", "----------")

	// 数据行
	for _, lCpu := range info {
		flags := strings.Join(lCpu.Flags, ",")
		if len(flags) > 22 {
			flags = flags[:19] + "..."
		}

		_, _ = fmt.Fprintf(w,
			colCPU+"\t"+colVendorID+"\t"+colFamily+"\t"+colModel+"\t"+
				colStepping+"\t"+colPhysicalID+"\t"+colCoreID+"\t"+colCores+"\t"+
				colModelName+"\t"+colMhz+"\t"+colCacheSize+"\t"+colFlags+"\t"+colMicrocode+"\n",
			lCpu.CPU,
			lCpu.VendorID,
			lCpu.Family,
			lCpu.Model,
			lCpu.Stepping,
			lCpu.PhysicalID,
			lCpu.CoreID,
			lCpu.Cores,
			lCpu.ModelName,
			lCpu.Mhz,
			lCpu.CacheSize,
			flags,
			lCpu.Microcode,
		)
	}

	_ = w.Flush()
}

type CPUPercentStat struct {
	CPU       string
	User      float64
	System    float64
	Idle      float64
	Nice      float64
	Iowait    float64
	Irq       float64
	Softirq   float64
	Steal     float64
	Guest     float64
	GuestNice float64
	Total     float64 // 非空闲时间总和（User + System + ...）
}

// getCPUPercent 获取每个 CPU 核心的使用率（%），类似 top
func getCPUPercent(interval time.Duration) ([]CPUPercentStat, error) {
	// 第一次采样
	first, err := cpu.TimesWithContext(context.Background(), true)
	if err != nil {
		return nil, err
	}

	time.Sleep(interval)

	// 第二次采样
	second, err := cpu.TimesWithContext(context.Background(), true)
	if err != nil {
		return nil, err
	}

	var result []CPUPercentStat

	for i, v := range second {
		// 找到对应的上一次数据
		prev := first[i]
		// 计算差值（单位：秒）
		user := v.User - prev.User
		system := v.System - prev.System
		idle := v.Idle - prev.Idle
		nice := v.Nice - prev.Nice
		iowait := v.Iowait - prev.Iowait
		irq := v.Irq - prev.Irq
		softirq := v.Softirq - prev.Softirq
		steal := v.Steal - prev.Steal
		guest := v.Guest - prev.Guest
		guestNice := v.GuestNice - prev.GuestNice

		// 总时间 = 所有时间之和
		total := user + system + idle + nice + iowait + irq + softirq + steal + guest + guestNice

		if total <= 0 {
			continue
		}

		// 转换为百分比
		percent := CPUPercentStat{
			CPU:       v.CPU,
			User:      (user / total) * 100,
			System:    (system / total) * 100,
			Idle:      (idle / total) * 100,
			Nice:      (nice / total) * 100,
			Iowait:    (iowait / total) * 100,
			Irq:       (irq / total) * 100,
			Softirq:   (softirq / total) * 100,
			Steal:     (steal / total) * 100,
			Guest:     (guest / total) * 100,
			GuestNice: (guestNice / total) * 100,
			Total:     ((user + system + nice + iowait + irq + softirq + steal) / total) * 100,
		}

		result = append(result, percent)
	}

	// 按 CPU 名称排序（cpu0, cpu1...）
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPU < result[j].CPU
	})

	return result, nil
}

// printCPUPercentTable 输出表格
func printCPUPercentTable(stats []CPUPercentStat) {
	w := tabwriter.NewWriter(os.Stdout, 10, 4, 3, ' ', 0)

	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"CPU", "User", "System", "Idle", "Nice", "Iowait", "Irq", "Softirq", "Steal", "Guest", "Total%")

	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"-----", "------", "--------", "--------", "--------", "--------", "---", "---------", "-------", "-------", "-------")

	for _, s := range stats {
		_, _ = fmt.Fprintf(w, "%s\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\n",
			s.CPU,
			s.User,
			s.System,
			s.Idle,
			s.Nice,
			s.Iowait,
			s.Irq,
			s.Softirq,
			s.Steal,
			s.Guest,
			s.Total,
		)
	}

	_ = w.Flush()
}
