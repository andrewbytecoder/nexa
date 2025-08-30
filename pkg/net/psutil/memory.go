package psutil

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/spf13/cobra"
	"log"
	"strings"
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
		log.Fatal(err)
		return
	}

	fmt.Println("-----------------------------------------------------------------------")
	if psMem.readable {
		fmt.Println("Free : " + HumanReadableBytesBinary(v.Free))
		fmt.Println("Used : " + HumanReadableBytesBinary(v.Used))
		fmt.Println("Total : " + HumanReadableBytesBinary(v.Total))
		fmt.Println("UsedPercent : " + HumanReadableBytesBinary(uint64(v.UsedPercent)))
		fmt.Println("CommittedAS : " + HumanReadableBytesBinary(v.CommittedAS))
		fmt.Println("CommitLimit : " + HumanReadableBytesBinary(v.CommitLimit))
		fmt.Println("VmallocTotal : " + HumanReadableBytesBinary(v.VmallocTotal))
		fmt.Println("VmallocUsed : " + HumanReadableBytesBinary(v.VmallocUsed))
		fmt.Println("VmallocChunk : " + HumanReadableBytesBinary(v.VmallocChunk))
		fmt.Println("HighFree : " + HumanReadableBytesBinary(v.HighFree))
		fmt.Println("HighTotal : " + HumanReadableBytesBinary(v.HighTotal))
		fmt.Println("LowFree : " + HumanReadableBytesBinary(v.LowFree))
		fmt.Println("LowTotal : " + HumanReadableBytesBinary(v.LowTotal))
		fmt.Println("Mapped : " + HumanReadableBytesBinary(v.Mapped))
		fmt.Println("Slab : " + HumanReadableBytesBinary(v.Slab))
		fmt.Println("Sreclaimable : " + HumanReadableBytesBinary(v.Sreclaimable))
		fmt.Println("Sunreclaim : " + HumanReadableBytesBinary(v.Sunreclaim))
		fmt.Println("WriteBack : " + HumanReadableBytesBinary(v.WriteBack))
		fmt.Println("WriteBackTmp : " + HumanReadableBytesBinary(v.WriteBackTmp))
		fmt.Println("PageTables : " + HumanReadableBytesBinary(v.PageTables))
		fmt.Println("Shared : " + HumanReadableBytesBinary(v.Shared))
		fmt.Println("HugePagesFree : " + HumanReadableBytesBinary(v.HugePagesFree))
		fmt.Println("HugePagesRsvd : " + HumanReadableBytesBinary(v.HugePagesRsvd))
		fmt.Println("HugePagesSurp : " + HumanReadableBytesBinary(v.HugePagesSurp))
		fmt.Println("HugePagesTotal : " + HumanReadableBytesBinary(v.HugePagesTotal))
		fmt.Println("HugePageSize : " + HumanReadableBytesBinary(v.HugePageSize))
		fmt.Println("AnonHugePages : " + HumanReadableBytesBinary(v.AnonHugePages))
	} else {

		fmt.Println(fmt.Sprintf("Free : %d", v.Free))
		fmt.Println(fmt.Sprintf("Used : %d", v.Used))
		fmt.Println(fmt.Sprintf("Total : %d", v.Total))
		fmt.Println(fmt.Sprintf("UsedPercent : %f", v.UsedPercent))
		fmt.Println(fmt.Sprintf("CommittedAS : %d", v.CommittedAS))
		fmt.Println(fmt.Sprintf("CommitLimit : %d", v.CommitLimit))
		fmt.Println(fmt.Sprintf("VmallocTotal : %d", v.VmallocTotal))
		fmt.Println(fmt.Sprintf("VmallocUsed : %d", v.VmallocUsed))
		fmt.Println(fmt.Sprintf("VmallocChunk : %d", v.VmallocChunk))
		fmt.Println(fmt.Sprintf("HighFree : %d", v.HighFree))
		fmt.Println(fmt.Sprintf("HighTotal : %d", v.HighTotal))
		fmt.Println(fmt.Sprintf("LowFree : %d", v.LowFree))
		fmt.Println(fmt.Sprintf("LowTotal : %d", v.LowTotal))
		fmt.Println(fmt.Sprintf("Mapped : %d", v.Mapped))
		fmt.Println(fmt.Sprintf("Slab : %d", v.Slab))
		fmt.Println(fmt.Sprintf("Sreclaimable : %d", v.Sreclaimable))
		fmt.Println(fmt.Sprintf("Sunreclaim : %d", v.Sunreclaim))
		fmt.Println(fmt.Sprintf("WriteBack : %d", v.WriteBack))
		fmt.Println(fmt.Sprintf("WriteBackTmp : %d", v.WriteBackTmp))
		fmt.Println(fmt.Sprintf("PageTables : %d", v.PageTables))
		fmt.Println(fmt.Sprintf("Shared : %d", v.Shared))
		fmt.Println(fmt.Sprintf("HugePagesFree : %d", v.HugePagesFree))
		fmt.Println(fmt.Sprintf("HugePagesRsvd : %d", v.HugePagesRsvd))
		fmt.Println(fmt.Sprintf("HugePagesSurp : %d", v.HugePagesSurp))
		fmt.Println(fmt.Sprintf("HugePagesTotal : %d", v.HugePagesTotal))
		fmt.Println(fmt.Sprintf("HugePageSize : %d", v.HugePageSize))
		fmt.Println(fmt.Sprintf("AnonHugePages : %d", v.AnonHugePages))
	}
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
	fmt.Println("-----------------------------------------------------------------------")
	s, err := mem.SwapDevices()
	if err != nil {
		log.Fatal(err)
		return
	}
	for _, v := range s {
		if psMem.readable {
			fmt.Println(fmt.Sprintf("Device : %s", v.Name))
			fmt.Println(fmt.Sprintf("Total : %s", HumanReadableBytesBinary(v.UsedBytes)))
			fmt.Println(fmt.Sprintf("Used : %s", HumanReadableBytesBinary(v.UsedBytes)))
		} else {
			fmt.Println(fmt.Sprintf("Device : %s", v.Name))
			fmt.Println(fmt.Sprintf("Total : %d", v.UsedBytes))
			fmt.Println(fmt.Sprintf("Used : %d", v.UsedBytes))
		}
	}

}

func (psMem *PsMem) showSwapInfo() {
	fmt.Println("-----------------------------------------------------------------------")
	s, err := mem.SwapMemory()
	if err != nil {
		log.Fatal(err)
		return
	}
	if psMem.readable {
		fmt.Println(fmt.Sprintf("Total : %s", HumanReadableBytesBinary(s.Total)))
		fmt.Println(fmt.Sprintf("Used : %s", HumanReadableBytesBinary(s.Used)))
		fmt.Println(fmt.Sprintf("Free : %s", HumanReadableBytesBinary(s.Free)))
		fmt.Println(fmt.Sprintf("UsedPercent : %f", s.UsedPercent))
		fmt.Println(fmt.Sprintf("Sin : %d", s.Sin))
		fmt.Println(fmt.Sprintf("Sout : %d", s.Sout))
		fmt.Println(fmt.Sprintf("PgIn : %d", s.PgIn))
		fmt.Println(fmt.Sprintf("PgOut : %d", s.PgOut))
		fmt.Println(fmt.Sprintf("PgFault : %d", s.PgFault))
		fmt.Println(fmt.Sprintf("PgMajFault : %d", s.PgMajFault))
	} else {
		fmt.Println(fmt.Sprintf("Total : %d", s.Total))
		fmt.Println(fmt.Sprintf("Used : %d", s.Used))
		fmt.Println(fmt.Sprintf("Free : %d", s.Free))
		fmt.Println(fmt.Sprintf("UsedPercent : %f", s.UsedPercent))
		fmt.Println(fmt.Sprintf("Sin : %d", s.Sin))
		fmt.Println(fmt.Sprintf("Sout : %d", s.Sout))
		fmt.Println(fmt.Sprintf("PgIn : %d", s.PgIn))
		fmt.Println(fmt.Sprintf("PgOut : %d", s.PgOut))
		fmt.Println(fmt.Sprintf("PgFault : %d", s.PgFault))
		fmt.Println(fmt.Sprintf("PgMajFault : %d", s.PgMajFault))
	}

}
