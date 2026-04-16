package psutil

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
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
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"CPU", "Percent"})
	if psCpu.perCpu {
		for i, p := range percents {
			_ = table.Append([]string{fmt.Sprintf("cpu%d", i), fmt.Sprintf("%.2f%%", p)})
		}
	} else if len(percents) > 0 {
		_ = table.Append([]string{"total", fmt.Sprintf("%.2f%%", percents[0])})
	}
	_ = table.Render()

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
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"CPU", "User", "System", "Idle", "Iowait", "Irq", "Softirq", "Steal", "Guest", "GuestNice", "Nice"})
	for _, s := range stats {
		_ = table.Append([]string{
			s.CPU,
			fmt.Sprintf("%.2f", s.User),
			fmt.Sprintf("%.2f", s.System),
			fmt.Sprintf("%.2f", s.Idle),
			fmt.Sprintf("%.2f", s.Iowait),
			fmt.Sprintf("%.2f", s.Irq),
			fmt.Sprintf("%.2f", s.Softirq),
			fmt.Sprintf("%.2f", s.Steal),
			fmt.Sprintf("%.2f", s.Guest),
			fmt.Sprintf("%.2f", s.GuestNice),
			fmt.Sprintf("%.2f", s.Nice),
		})
	}
	_ = table.Render()
}

// printCPUInfoTable 美观输出 CPU 信息表格
func printCPUInfoTable(info []cpu.InfoStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"CPU", "VendorID", "Family", "Model", "Stepping", "PhysicalID", "CoreID", "Cores", "ModelName", "Mhz", "CacheSize", "Flags", "Microcode"})
	for _, lCpu := range info {
		flags := strings.Join(lCpu.Flags, ",")
		if len(flags) > 22 {
			flags = flags[:19] + "..."
		}
		_ = table.Append([]string{
			fmt.Sprintf("%d", lCpu.CPU),
			lCpu.VendorID,
			lCpu.Family,
			lCpu.Model,
			fmt.Sprintf("%d", lCpu.Stepping),
			lCpu.PhysicalID,
			lCpu.CoreID,
			fmt.Sprintf("%d", lCpu.Cores),
			lCpu.ModelName,
			fmt.Sprintf("%.2f", lCpu.Mhz),
			fmt.Sprintf("%d", lCpu.CacheSize),
			flags,
			lCpu.Microcode,
		})
	}
	_ = table.Render()
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
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"CPU", "User", "System", "Idle", "Nice", "Iowait", "Irq", "Softirq", "Steal", "Guest", "Total%"})
	for _, s := range stats {
		_ = table.Append([]string{
			s.CPU,
			fmt.Sprintf("%.2f%%", s.User),
			fmt.Sprintf("%.2f%%", s.System),
			fmt.Sprintf("%.2f%%", s.Idle),
			fmt.Sprintf("%.2f%%", s.Nice),
			fmt.Sprintf("%.2f%%", s.Iowait),
			fmt.Sprintf("%.2f%%", s.Irq),
			fmt.Sprintf("%.2f%%", s.Softirq),
			fmt.Sprintf("%.2f%%", s.Steal),
			fmt.Sprintf("%.2f%%", s.Guest),
			fmt.Sprintf("%.2f%%", s.Total),
		})
	}
	_ = table.Render()
}
