package psutil

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
)

type PsDisk struct {
	psUtil *PsUtil

	// Command line flags.
	// Command line flags.
	readable      bool
	showType      string
	usagePath     string
	allPartitions bool
}

func NewPsDisk(psUtil *PsUtil) *PsDisk {
	return &PsDisk{
		psUtil: psUtil,
	}
}

const (
	tUsage     = "usage"
	tIOCounter = "IOCounter"
)

func (psDisk *PsDisk) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psDisk.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psDisk.showType, "type", "t", "all", strings.Join([]string{tAll, tUsage, tIOCounter}, "|"))
	cmd.Flags().StringVarP(&psDisk.usagePath, "path", "u", "", "if not set depend on -a")
	cmd.Flags().BoolVarP(&psDisk.allPartitions, "all", "a", true, "all partitions")
}

func (psDisk *PsDisk) GetDiskInfo() {
	if psDisk.showType == tAll || psDisk.showType == tUsage {
		psDisk.ShowUsage()
	}

	if psDisk.showType == tAll || psDisk.showType == tIOCounter {
		psDisk.ShowIOCounter()
	}
}

func (psDisk *PsDisk) ShowUsage() {
	allPartitionStat := make([]disk.PartitionStat, 0)
	if psDisk.usagePath == "" {
		allPartitionStat, _ = disk.Partitions(psDisk.allPartitions)
	} else {
		allPartitionStat = append(allPartitionStat, disk.PartitionStat{
			Mountpoint: psDisk.usagePath,
		})
	}
	mapUsage := make(map[string]*disk.UsageStat)
	for _, partitionStat := range allPartitionStat {
		if true == shouldSkipMount(partitionStat.Mountpoint, partitionStat.Fstype) {
			continue
		}
		usage, err := disk.Usage(partitionStat.Mountpoint)
		if err != nil {
			psDisk.psUtil.logger.Error("disk.Usage", zap.Error(err))
			continue
		}
		mapUsage[partitionStat.Mountpoint] = usage
	}

	err := PrintDiskUsageTable(mapUsage)
	if err != nil {
		psDisk.psUtil.logger.Error("PrintDiskUsageTable", zap.Error(err))
		return
	}
}

func (psDisk *PsDisk) ShowIOCounter() {
	allPartitionStat := make([]disk.PartitionStat, 0)
	if psDisk.usagePath == "" {
		allPartitionStat, _ = disk.Partitions(psDisk.allPartitions)
	} else {
		allPartitionStat = append(allPartitionStat, disk.PartitionStat{
			Mountpoint: psDisk.usagePath,
		})
	}
	var devices []string
	for _, partitionStat := range allPartitionStat {
		if true == shouldSkipMount(partitionStat.Mountpoint, partitionStat.Fstype) {
			continue
		}
		devices = append(devices, partitionStat.Device)
	}
	counters, err := disk.IOCounters(devices...)
	if err != nil {
		psDisk.psUtil.logger.Error("disk.IOCounters", zap.Error(err))
		return
	}
	psDisk.PrintIOCounter(counters)
}

func (psDisk *PsDisk) PrintIOCounter(counters map[string]disk.IOCountersStat) {

	table := tablewriter.NewWriter(os.Stdout)

	// 设置表头：每个字段名作为一列
	headers := []string{
		"Name", "ReadCount", "MergedReadCount", "WriteCount", "MergedWriteCount",
		"ReadBytes", "WriteBytes", "ReadTime", "WriteTime", "IopsInProgress",
		"IoTime", "WeightedIO", "SerialNumber", "Label",
	}
	table.Header(headers)
	for _, stat := range counters {
		// 添加数据行：每个字段值对应一列
		row := []string{
			stat.Name,
			fmt.Sprintf("%d", stat.ReadCount),
			fmt.Sprintf("%d", stat.MergedReadCount),
			fmt.Sprintf("%d", stat.WriteCount),
			fmt.Sprintf("%d", stat.MergedWriteCount),
			fmt.Sprintf("%d", stat.ReadBytes),
			fmt.Sprintf("%d", stat.WriteBytes),
			fmt.Sprintf("%d", stat.ReadTime),
			fmt.Sprintf("%d", stat.WriteTime),
			fmt.Sprintf("%d", stat.IopsInProgress),
			fmt.Sprintf("%d", stat.IoTime),
			fmt.Sprintf("%d", stat.WeightedIO),
			stat.SerialNumber,
			stat.Label,
		}
		err := table.Append(row)
		if err != nil {
			return
		}
	}
	err := table.Render()
	if err != nil {
		psDisk.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}

}

func shouldSkipMount(mountpoint, fstype string) bool {
	// 跳过用户运行时目录（包括 doc, gvfs）
	if strings.HasPrefix(mountpoint, "/run/user/") {
		return true
	}

	// 跳过 Docker 相关
	if strings.HasPrefix(mountpoint, "/var/lib/docker/") ||
		strings.HasPrefix(mountpoint, "/run/docker/") {
		return true
	}

	// 跳过 Snap
	if strings.HasPrefix(mountpoint, "/run/snapd/ns/") {
		return true
	}

	// 跳过媒体挂载（CD/DVD/USB）
	if strings.HasPrefix(mountpoint, "/media/") {
		return true
	}

	// 跳过临时/虚拟文件系统
	virtualTypes := []string{
		"nsfs", "overlay", "cgroup", "debugfs", "fusectl",
	}
	for _, t := range virtualTypes {
		if fstype == t {
			return true
		}
	}

	// 可选：跳过 FUSE 文件系统（如 gvfs, sshfs）
	if strings.HasPrefix(fstype, "fuse.") || fstype == "fuse" {
		return true
	}

	return false
}

// humanSize 将字节转换为可读单位（KB, MB, GB, TB）
func humanSize(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return strconv.FormatUint(bytes, 10)
	}
}

// humanInodes 将 inode 数量转为可读格式（如 1.2M）
func humanInodes(n uint64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	}
	return strconv.FormatUint(n, 10)
}

// PrintDiskUsageTable 输出单个路径的磁盘使用情况表格
func PrintDiskUsageTable(mapUsage map[string]*disk.UsageStat) error {

	// 使用 tabwriter 实现对齐
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// 表头
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"Filesystem", "Type", "Size", "Used", "Avail", "Use%", "Inodes", "IUse%", "Mounted on")

	// 分隔线
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 6),
		strings.Repeat("-", 6), strings.Repeat("-", 6), strings.Repeat("-", 5),
		strings.Repeat("-", 8), strings.Repeat("-", 6), strings.Repeat("-", 12))

	for mountOn, usage := range mapUsage {
		// 数据行
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.1f%%\t%s\t%.1f%%\t%s\n",
			usage.Fstype,                   // Filesystem（实际应为设备名，但 gopsutil 不直接提供）
			usage.Fstype,                   // Type（文件系统类型）
			humanSize(usage.Total),         // Size
			humanSize(usage.Used),          // Used
			humanSize(usage.Free),          // Avail
			usage.UsedPercent,              // Use%
			humanInodes(usage.InodesTotal), // Inodes
			usage.InodesUsedPercent,        // IUse%
			mountOn,                        // IUse%
		)

		fmt.Println("=============================path", usage.Path)
	}

	w.Flush()
	return nil
}
