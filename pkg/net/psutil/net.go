package psutil

import (
	"github.com/shirou/gopsutil/v4/net"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

type PsNet struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
	pernic   bool
}

func NewPsnet(psUtil *PsUtil) *PsNet {
	return &PsNet{
		psUtil: psUtil,
	}
}

func (psnet *PsNet) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psnet.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psnet.showType, "type", "t", "all", strings.Join([]string{tAll}, "|"))
	cmd.Flags().BoolVarP(&psnet.pernic, "pernic", "p", true, "If pernic argument is false, return only sum of all information")
}

func (psNet *PsNet) GetnetInfo() {
	if psNet.showType == tAll || psNet.showType == "" {
	}

	if psNet.showType == tAll || psNet.showType == "" {
	}
}

func (psNet *PsNet) shownetAvg() {
	counters, err := net.IOCounters(psNet.pernic)
	if err != nil {
		psNet.psUtil.logger.Error("Error getting network info", zap.Error(err))
		return
	}
	net.PrintIOCounter(counters)
}
