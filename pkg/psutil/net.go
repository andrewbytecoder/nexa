package psutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	net "github.com/shirou/gopsutil/v4/net"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type PsNet struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
	pid      int32
	kind     string
	pernic   bool
	perCpu   bool
}

func NewPsnet(psUtil *PsUtil) *PsNet {
	return &PsNet{
		psUtil: psUtil,
	}
}

func (psnet *PsNet) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psnet.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psnet.showType, "type", "t", "all", strings.Join([]string{tAll, tNetIfConfig, tNetIOCounter,
		tNetConnections, tNetConntrack, tNetPids}, "|"))
	cmd.Flags().Int32VarP(&psnet.pid, "pid", "p", 0, "get connections by pid")
	cmd.Flags().StringVarP(&psnet.kind, "kind", "k", "all", strings.Join([]string{"all", "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6",
		"unix", "inet", "inet4", "inet6"}, "|"))
	cmd.Flags().BoolVarP(&psnet.pernic, "pernic", "n", false, " If pernic argument is false, return only sum of all information")
	cmd.Flags().BoolVarP(&psnet.perCpu, "percpu", "c", false, "If 'percpu' is false, the result will contain exactly one item with totals/summary")
}

func (psNet *PsNet) GetnetInfo() {
	if psNet.showType == tAll || psNet.showType == tNetIfConfig {
		psNet.showNetIfConfig()
	}

	if psNet.showType == tAll || psNet.showType == tNetIOCounter {
		psNet.showNetIOCounters()
	}

	if psNet.showType == tAll || psNet.showType == tNetConnections {
		psNet.showNetConnections()
	}

	if psNet.showType == tAll || psNet.showType == tNetConntrack {
		psNet.showNetConntrack()
	}

	if psNet.showType == tAll || psNet.showType == tNetPids {
		psNet.showNetPids()
	}

}

func (psNet *PsNet) showNetPids() {
	pids, err := net.Pids()
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	fmt.Printf("PIDs: %v\n", pids)
}

func (psNet *PsNet) showNetConntrack() {
	stats, err := net.ConntrackStats(psNet.perCpu)
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	// 构建表头
	headers := []string{"Metric"}
	for i := range stats {
		headers = append(headers, fmt.Sprintf("CPU %d", i))
	}

	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header(headers)
	// 所有指标名称（按顺序）
	metrics := []struct {
		name string
		get  func(net.ConntrackStat) uint32
	}{
		{"Entries", func(s net.ConntrackStat) uint32 { return s.Entries }},
		{"Searched", func(s net.ConntrackStat) uint32 { return s.Searched }},
		{"Found", func(s net.ConntrackStat) uint32 { return s.Found }},
		{"New", func(s net.ConntrackStat) uint32 { return s.New }},
		{"Invalid", func(s net.ConntrackStat) uint32 { return s.Invalid }},
		{"Ignore", func(s net.ConntrackStat) uint32 { return s.Ignore }},
		{"Delete", func(s net.ConntrackStat) uint32 { return s.Delete }},
		{"DeleteList", func(s net.ConntrackStat) uint32 { return s.DeleteList }},
		{"Insert", func(s net.ConntrackStat) uint32 { return s.Insert }},
		{"InsertFailed", func(s net.ConntrackStat) uint32 { return s.InsertFailed }},
		{"Drop", func(s net.ConntrackStat) uint32 { return s.Drop }},
		{"EarlyDrop", func(s net.ConntrackStat) uint32 { return s.EarlyDrop }},
		{"IcmpError", func(s net.ConntrackStat) uint32 { return s.IcmpError }},
		{"ExpectNew", func(s net.ConntrackStat) uint32 { return s.ExpectNew }},
		{"ExpectCreate", func(s net.ConntrackStat) uint32 { return s.ExpectCreate }},
		{"ExpectDelete", func(s net.ConntrackStat) uint32 { return s.ExpectDelete }},
		{"SearchRestart", func(s net.ConntrackStat) uint32 { return s.SearchRestart }},
	}

	// 添加每一行（每个 metric）
	for _, m := range metrics {
		row := []string{m.name}
		for _, s := range stats {
			row = append(row, formatUint32(m.get(s)))
		}
		err := table.Append(row)
		if err != nil {
			psNet.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err = table.Render()
	if err != nil {
		psNet.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

// formatUint32 添加千位分隔符并右对齐
func formatUint32(n uint32) string {
	return fmt.Sprintf("%12s", comma(n))
}

// comma 添加千位分隔符
func comma(n uint32) string {
	in := strconv.FormatUint(uint64(n), 10)
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}
func (psNet *PsNet) showNetConnections() {
	connections, err := net.ConnectionsPid(psNet.kind, psNet.pid)
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"FD", "Family", "Type", "Local Addr", "Remote Addr", "Status", "UIDs", "PID"})
	// 添加每一行
	for _, conn := range connections {
		row := []string{
			fmt.Sprintf("%d", conn.Fd),
			fmt.Sprintf("%d", conn.Family),
			fmt.Sprintf("%d", conn.Type),
			conn.Laddr.String(),
			conn.Raddr.String(),
			conn.Status,
			formatUids(conn.Uids),
			fmt.Sprintf("%d", conn.Pid),
		}
		err := table.Append(row)
		if err != nil {
			psNet.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err = table.Render()
	if err != nil {
		psNet.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

// formatUids 将 UID 列表转为字符串
func formatUids(uids []int32) string {
	if len(uids) == 0 {
		return ""
	}
	strs := make([]string, len(uids))
	for i, uid := range uids {
		strs[i] = strconv.Itoa(int(uid))
	}
	return strings.Join(strs, ",")
}
func (psNet *PsNet) showNetIfConfig() {
	inters, err := net.Interfaces()
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Name", "Index", "MTU", "Hardware Addr", "Flags", "Addresses"})

	// 添加每一行
	for _, iface := range inters {
		row := []string{
			iface.Name,
			fmt.Sprintf("%d", iface.Index),
			fmt.Sprintf("%d", iface.MTU),
			iface.HardwareAddr,
			strings.Join(iface.Flags, ","),       // 列表转字符串
			InterFaceAddrListString(iface.Addrs), // 多地址换行显示
		}
		err := table.Append(row)
		if err != nil {
			psNet.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err = table.Render()
	if err != nil {
		psNet.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

// InterFaceAddrListString 方法，便于表格输出
func InterFaceAddrListString(ial net.InterfaceAddrList) string {
	var addrs []string
	for _, addr := range ial {
		addrs = append(addrs, addr.Addr)
	}
	return strings.Join(addrs, "\n")
}

func (psNet *PsNet) showNetIOCounters() {
	counters, err := net.IOCounters(psNet.pernic)
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	psNet.printIOCounters(counters)
}

// formatUint64 添加千位分隔符（例如 1234567 -> 1,234,567）
func formatUint64(n uint64) string {
	if n == 0 {
		return "0"
	}
	out := ""
	s := fmt.Sprintf("%d", n)
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(r)
	}
	return out
}

func (psNet *PsNet) printIOCounters(stats []net.IOCountersStat) {
	// Configure colors: green headers, cyan/magenta rows, yellow footer
	colorCfg := renderer.ColorizedConfig{
		Header: renderer.Tint{
			FG: renderer.Colors{color.FgGreen, color.Bold}, // Green bold headers
			BG: renderer.Colors{color.BgHiWhite},
		},
		Column: renderer.Tint{
			FG: renderer.Colors{color.FgCyan}, // Default cyan for rows
			Columns: []renderer.Tint{
				{FG: renderer.Colors{color.FgMagenta}}, // Magenta for column 0
				{},                                     // Inherit default (cyan)
				{FG: renderer.Colors{color.FgHiRed}},   // High-intensity red for column 2
			},
		},
		Footer: renderer.Tint{
			FG: renderer.Colors{color.FgYellow, color.Bold}, // Yellow bold footer
			Columns: []renderer.Tint{
				{},                                      // Inherit default
				{FG: renderer.Colors{color.FgHiYellow}}, // High-intensity yellow for column 1
				{},                                      // Inherit default
			},
		},
		Border:    renderer.Tint{FG: renderer.Colors{color.FgWhite}}, // White borders
		Separator: renderer.Tint{FG: renderer.Colors{color.FgWhite}}, // White separators
	}

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewColorized(colorCfg)),
		tablewriter.WithConfig(tablewriter.Config{
			Row: tw.CellConfig{
				Formatting:   tw.CellFormatting{AutoWrap: tw.WrapNormal}, // Wrap long content
				Alignment:    tw.CellAlignment{Global: tw.AlignRight},    // Left-align rows
				ColMaxWidths: tw.CellWidth{Global: 25},
			},
			Footer: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignRight},
			},
		}),
	)
	// 创建表格
	table.Header([]string{
		"Name", "Bytes Sent", "Bytes Received",
		"Packets Sent", "Packets Received",
		"ErrIn", "ErrOut", "DropIn", "DropOut", "FIFOIn", "FIFOOut",
	})

	// 添加每一行数据
	for _, s := range stats {
		row := []string{
			s.Name,
			formatUint64(s.BytesSent),
			formatUint64(s.BytesRecv),
			formatUint64(s.PacketsSent),
			formatUint64(s.PacketsRecv),
			fmt.Sprintf("%d", s.Errin),
			fmt.Sprintf("%d", s.Errout),
			fmt.Sprintf("%d", s.Dropin),
			fmt.Sprintf("%d", s.Dropout),
			fmt.Sprintf("%d", s.Fifoin),
			fmt.Sprintf("%d", s.Fifoout),
		}
		err := table.Append(row)
		if err != nil {
			psNet.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err := table.Render()
	if err != nil {
		psNet.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}
