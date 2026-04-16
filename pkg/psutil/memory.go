package psutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type PsMem struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
}

func NewPsMem(psUtil *PsUtil) *PsMem {
	psMem := &PsMem{
		psUtil: psUtil,
	}
	return psMem
}

func (psMem *PsMem) ParseFlags(cmd *cobra.Command) {
	//  -h, --human-readable
	cmd.Flags().BoolVarP(&psMem.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psMem.showType, "type", "t", "all", strings.Join([]string{tAll, tMem, tSwap, tSwapDev}, "|"))
}

func (psMem *PsMem) GetMemInfo() {
	if psMem.showType == tAll || psMem.showType == tMem {
		psMem.showMemInfo()
	}

	if psMem.showType == tAll || psMem.showType == tSwap {
		psMem.showSwapDevInfo()
	}

	if psMem.showType == tAll || psMem.showType == tSwapDev {
		psMem.showSwapInfo()
	}
}

func (psMem *PsMem) showMemInfo() {

	v, err := mem.VirtualMemory()
	if err != nil {
		psMem.psUtil.logger.Error("mem.VirtualMemory", zap.Error(err))
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})
	if psMem.readable {
		_ = table.Append([]string{"Free", HumanReadableBytesBinary(v.Free)})
		_ = table.Append([]string{"Used", HumanReadableBytesBinary(v.Used)})
		_ = table.Append([]string{"Total", HumanReadableBytesBinary(v.Total)})
		_ = table.Append([]string{"UsedPercent", fmt.Sprintf("%.2f%%", v.UsedPercent)})
		_ = table.Append([]string{"CommittedAS", HumanReadableBytesBinary(v.CommittedAS)})
		_ = table.Append([]string{"CommitLimit", HumanReadableBytesBinary(v.CommitLimit)})
		_ = table.Append([]string{"VmallocTotal", HumanReadableBytesBinary(v.VmallocTotal)})
		_ = table.Append([]string{"VmallocUsed", HumanReadableBytesBinary(v.VmallocUsed)})
		_ = table.Append([]string{"VmallocChunk", HumanReadableBytesBinary(v.VmallocChunk)})
		_ = table.Append([]string{"HighFree", HumanReadableBytesBinary(v.HighFree)})
		_ = table.Append([]string{"HighTotal", HumanReadableBytesBinary(v.HighTotal)})
		_ = table.Append([]string{"LowFree", HumanReadableBytesBinary(v.LowFree)})
		_ = table.Append([]string{"LowTotal", HumanReadableBytesBinary(v.LowTotal)})
		_ = table.Append([]string{"Mapped", HumanReadableBytesBinary(v.Mapped)})
		_ = table.Append([]string{"Slab", HumanReadableBytesBinary(v.Slab)})
		_ = table.Append([]string{"Sreclaimable", HumanReadableBytesBinary(v.Sreclaimable)})
		_ = table.Append([]string{"Sunreclaim", HumanReadableBytesBinary(v.Sunreclaim)})
		_ = table.Append([]string{"WriteBack", HumanReadableBytesBinary(v.WriteBack)})
		_ = table.Append([]string{"WriteBackTmp", HumanReadableBytesBinary(v.WriteBackTmp)})
		_ = table.Append([]string{"PageTables", HumanReadableBytesBinary(v.PageTables)})
		_ = table.Append([]string{"Shared", HumanReadableBytesBinary(v.Shared)})
		_ = table.Append([]string{"HugePagesFree", HumanReadableBytesBinary(v.HugePagesFree)})
		_ = table.Append([]string{"HugePagesRsvd", HumanReadableBytesBinary(v.HugePagesRsvd)})
		_ = table.Append([]string{"HugePagesSurp", HumanReadableBytesBinary(v.HugePagesSurp)})
		_ = table.Append([]string{"HugePagesTotal", HumanReadableBytesBinary(v.HugePagesTotal)})
		_ = table.Append([]string{"HugePageSize", HumanReadableBytesBinary(v.HugePageSize)})
		_ = table.Append([]string{"AnonHugePages", HumanReadableBytesBinary(v.AnonHugePages)})
	} else {
		_ = table.Append([]string{"Free", fmt.Sprintf("%d", v.Free)})
		_ = table.Append([]string{"Used", fmt.Sprintf("%d", v.Used)})
		_ = table.Append([]string{"Total", fmt.Sprintf("%d", v.Total)})
		_ = table.Append([]string{"UsedPercent", fmt.Sprintf("%.2f%%", v.UsedPercent)})
		_ = table.Append([]string{"CommittedAS", fmt.Sprintf("%d", v.CommittedAS)})
		_ = table.Append([]string{"CommitLimit", fmt.Sprintf("%d", v.CommitLimit)})
		_ = table.Append([]string{"VmallocTotal", fmt.Sprintf("%d", v.VmallocTotal)})
		_ = table.Append([]string{"VmallocUsed", fmt.Sprintf("%d", v.VmallocUsed)})
		_ = table.Append([]string{"VmallocChunk", fmt.Sprintf("%d", v.VmallocChunk)})
		_ = table.Append([]string{"HighFree", fmt.Sprintf("%d", v.HighFree)})
		_ = table.Append([]string{"HighTotal", fmt.Sprintf("%d", v.HighTotal)})
		_ = table.Append([]string{"LowFree", fmt.Sprintf("%d", v.LowFree)})
		_ = table.Append([]string{"LowTotal", fmt.Sprintf("%d", v.LowTotal)})
		_ = table.Append([]string{"Mapped", fmt.Sprintf("%d", v.Mapped)})
		_ = table.Append([]string{"Slab", fmt.Sprintf("%d", v.Slab)})
		_ = table.Append([]string{"Sreclaimable", fmt.Sprintf("%d", v.Sreclaimable)})
		_ = table.Append([]string{"Sunreclaim", fmt.Sprintf("%d", v.Sunreclaim)})
		_ = table.Append([]string{"WriteBack", fmt.Sprintf("%d", v.WriteBack)})
		_ = table.Append([]string{"WriteBackTmp", fmt.Sprintf("%d", v.WriteBackTmp)})
		_ = table.Append([]string{"PageTables", fmt.Sprintf("%d", v.PageTables)})
		_ = table.Append([]string{"Shared", fmt.Sprintf("%d", v.Shared)})
		_ = table.Append([]string{"HugePagesFree", fmt.Sprintf("%d", v.HugePagesFree)})
		_ = table.Append([]string{"HugePagesRsvd", fmt.Sprintf("%d", v.HugePagesRsvd)})
		_ = table.Append([]string{"HugePagesSurp", fmt.Sprintf("%d", v.HugePagesSurp)})
		_ = table.Append([]string{"HugePagesTotal", fmt.Sprintf("%d", v.HugePagesTotal)})
		_ = table.Append([]string{"HugePageSize", fmt.Sprintf("%d", v.HugePageSize)})
		_ = table.Append([]string{"AnonHugePages", fmt.Sprintf("%d", v.AnonHugePages)})
	}
	_ = table.Render()
}

// HumanReadableBytesBinary 使用二进制单位（GiB, MiB）
func HumanReadableBytesBinary(bytes uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	base := 1024.0
	i := 0
	for ; i < len(units)-1 && value >= base; i++ {
		value /= base
	}

	return fmt.Sprintf("%.2f %s", value, units[i])
}

func (psMem *PsMem) showSwapDevInfo() {
	s, err := mem.SwapDevices()
	if err != nil {
		psMem.psUtil.logger.Error("mem.SwapDevices", zap.Error(err))
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Device", "Total", "Used"})
	for _, v := range s {
		if psMem.readable {
			_ = table.Append([]string{
				v.Name,
				HumanReadableBytesBinary(v.UsedBytes),
				HumanReadableBytesBinary(v.UsedBytes),
			})
		} else {
			_ = table.Append([]string{
				v.Name,
				fmt.Sprintf("%d", v.UsedBytes),
				fmt.Sprintf("%d", v.UsedBytes),
			})
		}
	}
	_ = table.Render()

}

func (psMem *PsMem) showSwapInfo() {
	s, err := mem.SwapMemory()
	if err != nil {
		psMem.psUtil.logger.Error("mem.SwapMemory", zap.Error(err))
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})
	if psMem.readable {
		_ = table.Append([]string{"Total", HumanReadableBytesBinary(s.Total)})
		_ = table.Append([]string{"Used", HumanReadableBytesBinary(s.Used)})
		_ = table.Append([]string{"Free", HumanReadableBytesBinary(s.Free)})
	} else {
		_ = table.Append([]string{"Total", fmt.Sprintf("%d", s.Total)})
		_ = table.Append([]string{"Used", fmt.Sprintf("%d", s.Used)})
		_ = table.Append([]string{"Free", fmt.Sprintf("%d", s.Free)})
	}
	_ = table.Append([]string{"UsedPercent", fmt.Sprintf("%.2f%%", s.UsedPercent)})
	_ = table.Append([]string{"Sin", fmt.Sprintf("%d", s.Sin)})
	_ = table.Append([]string{"Sout", fmt.Sprintf("%d", s.Sout)})
	_ = table.Append([]string{"PgIn", fmt.Sprintf("%d", s.PgIn)})
	_ = table.Append([]string{"PgOut", fmt.Sprintf("%d", s.PgOut)})
	_ = table.Append([]string{"PgFault", fmt.Sprintf("%d", s.PgFault)})
	_ = table.Append([]string{"PgMajFault", fmt.Sprintf("%d", s.PgMajFault)})
	_ = table.Render()

}
