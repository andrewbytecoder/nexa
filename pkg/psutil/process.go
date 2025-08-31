package psutil

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"sort"
	"strconv"
	"strings"
)

type PsProgress struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
	pid      int32
}

func NewPsProcess(psUtil *PsUtil) *PsProgress {
	return &PsProgress{
		psUtil: psUtil,
	}
}

func (psProgress *PsProgress) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psProgress.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psProgress.showType, "type", "t", "all", strings.Join([]string{tAll, tProcess, tProcessByPid}, "|"))
	cmd.Flags().Int32VarP(&psProgress.pid, "pid", "p", 0, "process id")
}

func (psProgress *PsProgress) GetProcessInfo() {
	if psProgress.showType == tAll || psProgress.showType == tProcess {
		psProgress.showProgress()
	}

	if psProgress.showType == tAll || psProgress.showType == tProcessByPid {
		psProgress.GetProcessInfoByPid()
	}
}

func (psProgress *PsProgress) GetProcessInfoByPid() {

	exists, err := process.PidExists(psProgress.pid)
	if err != nil || !exists {
		fmt.Printf("process %d not exists\n", psProgress.pid)
		return
	}
	processes, err := process.Processes()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
		return
	}
	var lpProcess *process.Process
	for _, processInfo := range processes {
		if processInfo.Pid == psProgress.pid {
			lpProcess = processInfo
			break
		}
	}

	if lpProcess == nil {
		fmt.Printf("process %d not exists\n", psProgress.pid)
		return
	}
	fmt.Println("--------------------------------- process with pid ------------------------------------")
	times, err := lpProcess.Times()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printTimesStatTable(times)
	threadsTimes, err := lpProcess.Threads()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	psProgress.printCPUTimesTable(threadsTimes)

	switches, err := lpProcess.NumCtxSwitches()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printSwitchesTable(switches)

	memInfo, err := lpProcess.MemoryInfo()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printMemoryInfoTable(memInfo)

	counters, err := lpProcess.IOCounters()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printIOCountersTable(counters)
	connections, err := lpProcess.Connections()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printConnectionTable(connections)

	files, err := lpProcess.OpenFiles()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
	}
	printOpenFilesTable(files)

}

// printOpenFilesTable 输出 []OpenFilesStat 表格
func printOpenFilesTable(files []process.OpenFilesStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"FD", "PATH"})

	for _, f := range files {
		row := []string{
			strconv.FormatUint(f.Fd, 10),
			f.Path,
		}
		table.Append(row)
	}

	table.Render()
}

// formatAddr 格式化地址，处理通配符和 IPv6
func formatAddr(addr net.Addr) string {
	if addr.IP == "" {
		return "*:*"
	}
	ip := addr.IP
	port := strconv.FormatUint(uint64(addr.Port), 10)

	// IPv6 处理
	if contains(ip, ":") {
		return fmt.Sprintf("[%s]:%s", ip, port)
	}
	return fmt.Sprintf("%s:%s", ip, port)
}

// familyToString 转换 Family 数值为可读字符串
func familyToString(f uint32) string {
	switch f {
	case 1:
		return "UNIX"
	case 2:
		return "IPv4"
	case 10:
		return "IPv6"
	default:
		return fmt.Sprintf("%d", f)
	}
}

// typeToString 转换 Type 数值为可读字符串
func typeToString(t uint32) string {
	switch t {
	case 1:
		return "TCP"
	case 2:
		return "UDP"
	case 3:
		return "UDPLITE"
	case 5:
		return "RAW"
	default:
		return fmt.Sprintf("%d", t)
	}
}

// printConnectionTable 输出 []ConnectionStat 表格
func printConnectionTable(conns []net.ConnectionStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"FD", "FAMILY", "TYPE", "LOCAL", "REMOTE", "STATUS", "PID"})

	for _, c := range conns {
		row := []string{
			strconv.FormatUint(uint64(c.Fd), 10),
			familyToString(c.Family),
			typeToString(c.Type),
			formatAddr(c.Laddr),
			formatAddr(c.Raddr),
			c.Status,
			strconv.FormatInt(int64(c.Pid), 10),
		}
		table.Append(row)
	}

	table.Render()
}

// contains 简单字符串包含判断（替代 strings.Contains）
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// formatCount 格式化计数，添加千分位逗号
func formatCount(n uint64) string {
	in := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return fmt.Sprintf("%12s", out)
}

// printIOCountersTable 输出 *IOCountersStat 表格
func printIOCountersTable(io *process.IOCountersStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})

	data := [][]string{
		{"Read Count", formatCount(io.ReadCount)},
		{"Write Count", formatCount(io.WriteCount)},
		{"Read Bytes", formatBytes(io.ReadBytes)},
		{"Write Bytes", formatBytes(io.WriteBytes)},
		{"Disk Read Bytes", formatBytes(io.DiskReadBytes)},
		{"Disk Write Bytes", formatBytes(io.DiskWriteBytes)},
	}

	for _, row := range data {
		table.Append(row)
	}

	table.Render()
}

// formatBytes 将字节转换为可读格式（自动选择单位）
func formatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%8.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%8.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%8.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%8.2f B", float64(bytes))
	}
}

// printMemoryInfoTable 输出 *MemoryInfoStat 表格
func printMemoryInfoTable(mem *process.MemoryInfoStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})

	data := [][]string{
		{"RSS", formatBytes(mem.RSS)},
		{"VMS", formatBytes(mem.VMS)},
		{"HWM", formatBytes(mem.HWM)},
		{"Data", formatBytes(mem.Data)},
		{"Stack", formatBytes(mem.Stack)},
		{"Locked", formatBytes(mem.Locked)},
		{"Swap", formatBytes(mem.Swap)},
	}

	for _, row := range data {
		table.Append(row)
	}

	table.Render()
}
func printSwitchesTable(switches *process.NumCtxSwitchesStat) {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})

	data := [][]string{
		{"Voluntary", formatInt(switches.Voluntary)},
		{"Involuntary", formatInt(switches.Involuntary)},
	}

	for _, row := range data {
		table.Append(row)
	}

	table.Render()
}

// printTimesStatTable 输出 *TimesStat 表格
func printTimesStatTable(ts *cpu.TimesStat) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})
	// 格式化浮点数，保留 1 位小数，右对齐
	format := func(v float64) string {
		return fmt.Sprintf("%12.1f", v)
	}

	data := [][]string{
		{"CPU", ts.CPU},
		{"User(s)", format(ts.User)},
		{"System(s)", format(ts.System)},
		{"Idle(s)", format(ts.Idle)},
		{"Nice(s)", format(ts.Nice)},
		{"Iowait(s)", format(ts.Iowait)},
		{"Irq(s)", format(ts.Irq)},
		{"Softirq(s)", format(ts.Softirq)},
		{"Steal(s)", format(ts.Steal)},
		{"Guest(s)", format(ts.Guest)},
		{"GuestNice(s)", format(ts.GuestNice)},
	}

	for _, v := range data {
		table.Append(v)
	}

	table.Render()
}

// formatFloat 格式化浮点数，保留1位小数，右对齐
func formatFloat(f float64) string {
	return fmt.Sprintf("%8.1f", f)
}

// printCPUTimesTable 输出 map[int32]*TimesStat 表格
func (psProgress *PsProgress) printCPUTimesTable(cpuTimes map[int32]*cpu.TimesStat) {
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"CPU", "User(s)", "System(s)", "Idle(s)", "Nice(s)", "Iowait(s)", "Irq(s)", "Softirq(s)", "Steal(s)", "Guest(s)", "GuestNice(s)"})

	// 提取 key 并排序（保证输出顺序）
	var keys []int32
	for k := range cpuTimes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j] // 按 CPU ID 排序
	})

	// 填充表格
	for _, k := range keys {
		ts := cpuTimes[k]
		cpuName := ts.CPU
		if cpuName == "" {
			if k == -1 {
				cpuName = "total"
			} else {
				cpuName = "cpu" + strconv.Itoa(int(k))
			}
		}

		row := []string{
			cpuName,
			formatFloat(ts.User),
			formatFloat(ts.System),
			formatFloat(ts.Idle),
			formatFloat(ts.Nice),
			formatFloat(ts.Iowait),
			formatFloat(ts.Irq),
			formatFloat(ts.Softirq),
			formatFloat(ts.Steal),
			formatFloat(ts.Guest),
			formatFloat(ts.GuestNice),
		}
		err := table.Append(row)
		if err != nil {
			psProgress.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err := table.Render()
	if err != nil {
		psProgress.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

func (psProgress *PsProgress) showProgress() {
	processes, err := process.Processes()
	if err != nil {
		psProgress.psUtil.logger.Error(err.Error())
		return
	}
	psProgress.printProcessTable(processes)
}

// formatMB 将字节转为 MB，保留 1 位小数
func formatMB(bytes uint64) string {
	mb := float64(bytes) / 1024 / 1024
	return fmt.Sprintf("%10.1f", mb)
}

// formatInt 格式化整数（右对齐）
func formatInt(n int64) string {
	return fmt.Sprintf("%12s", comma(uint32(n)))
}

// printProcessTable 输出 []*Process 表格
func (psProgress *PsProgress) printProcessTable(processes []*process.Process) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"PID", "PPID", "Name", "Status", "Threads", "RSS(MB)", "VMS(MB)", "CPU User(s)", "Voluntary CS"})

	for _, p := range processes {
		name, _ := p.Name()
		statuVec, _ := p.Status()
		status := strings.Join(statuVec, "|")
		numThreads, _ := p.NumThreads()
		memInfo, _ := p.MemoryInfo()
		userName, _ := p.Username()
		numCtxSwitches, _ := p.NumCtxSwitches()
		ppid, _ := p.Ppid()

		row := []string{
			strconv.Itoa(int(p.Pid)),
			strconv.Itoa(int(ppid)),
			truncate(name, 16), // 限制长度
			status,
			strconv.Itoa(int(numThreads)),
			formatMB(memInfo.RSS),
			formatMB(memInfo.VMS),
			userName,
			formatInt(numCtxSwitches.Voluntary),
		}
		err := table.Append(row)
		if err != nil {
			psProgress.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err := table.Render()
	if err != nil {
		psProgress.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

// truncate 字符串截断
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
