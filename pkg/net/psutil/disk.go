package psutil

import (
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

type PsDisk struct {
	psUtil *PsUtil

	// Command line flags.
	// Command line flags.
	readable  bool
	showType  string
	usagePath string
}

func NewPsDisk(psUtil *PsUtil) *PsDisk {
	return &PsDisk{
		psUtil: psUtil,
	}
}

const ()

func (psDisk *PsDisk) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psDisk.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psDisk.showType, "type", "t", "all", strings.Join([]string{tAll, tTimes, tInfo}, "|"))
	cmd.Flags().StringVarP(&psDisk.usagePath, "usage-path", "u", "", "usage path")

}

func (psDisk *PsDisk) GetCpuInfo() {
	if psDisk.showType == tAll || psDisk.showType == tTimes {
	}

	if psDisk.showType == tAll || psDisk.showType == tInfo {
	}

}

func (psDisk *PsDisk) ShowUsage() {
	usage, err := disk.Usage("")
	if err != nil {
		psDisk.psUtil.logger.Error("disk.Usage", zap.Error(err))
		return
	}
}
