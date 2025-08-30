package psutil

import (
	"github.com/shirou/gopsutil/v4/net"
	"github.com/spf13/cobra"
	"strings"
)

type PsNet struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
}

func NewPsnet(psUtil *PsUtil) *PsNet {
	return &PsNet{
		psUtil: psUtil,
	}
}

func (psnet *PsNet) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psnet.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psnet.showType, "type", "t", "all", strings.Join([]string{tAll}, "|"))
}

func (psnet *PsNet) GetnetInfo() {
	if psnet.showType == tAll || psnet.showType == "" {
	}

	if psnet.showType == tAll || psnet.showType == "" {
	}
}

func (psnet *PsNet) shownetAvg() {

	net.IOCounters(true)
}
