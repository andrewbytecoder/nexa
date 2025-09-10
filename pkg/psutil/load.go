package psutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type PsLoad struct {
	psUtil *PsUtil

	// Command line flags.
	readable bool
	showType string
}

func NewPsLoad(psUtil *PsUtil) *PsLoad {
	return &PsLoad{
		psUtil: psUtil,
	}
}

func (psLoad *PsLoad) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&psLoad.readable, "human-readable", "H", true, "human readable output")
	cmd.Flags().StringVarP(&psLoad.showType, "type", "t", "all", strings.Join([]string{tAll, tLoadAvg, tLoadMisc}, "|"))
}

func (psLoad *PsLoad) GetLoadInfo() {
	if psLoad.showType == tAll || psLoad.showType == tLoadAvg {
		psLoad.showLoadAvg()
	}

	if psLoad.showType == tAll || psLoad.showType == tLoadMisc {
		psLoad.showLoadMisc()
	}
}

func (psLoad *PsLoad) showLoadAvg() {
	loadAvg, err := load.Avg()
	if err != nil {
		psLoad.psUtil.logger.Error("get load avg error", zap.Error(err))
		return
	}
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Load1", "Load5", "Load15"})

	// 添加数据行
	row := []string{
		fmt.Sprintf("%.2f", loadAvg.Load1),
		fmt.Sprintf("%.2f", loadAvg.Load5),
		fmt.Sprintf("%.2f", loadAvg.Load15),
	}
	err = table.Append(row)
	if err != nil {
		psLoad.psUtil.logger.Error("table.Append", zap.Error(err))
		return
	}

	err = table.Render()
	if err != nil {
		psLoad.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}

func (psLoad *PsLoad) showLoadMisc() {
	loadMisc, err := load.Misc()
	if err != nil {
		psLoad.psUtil.logger.Error("get load avg error", zap.Error(err))
		return
	}
	// 创建表格
	// 创建表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Metric", "Value"})

	// 添加数据行
	data := [][]string{
		{"Total Processes", fmt.Sprintf("%d", loadMisc.ProcsTotal)},
		{"Processes Created", fmt.Sprintf("%d", loadMisc.ProcsCreated)},
		{"Running Processes", fmt.Sprintf("%d", loadMisc.ProcsRunning)},
		{"Blocked Processes", fmt.Sprintf("%d", loadMisc.ProcsBlocked)},
		{"Context Switches", fmt.Sprintf("%d", loadMisc.Ctxt)},
	}

	for _, v := range data {
		err := table.Append(v)
		if err != nil {
			psLoad.psUtil.logger.Error("table.Append", zap.Error(err))
			return
		}
	}

	err = table.Render()
	if err != nil {
		psLoad.psUtil.logger.Error("table.Render", zap.Error(err))
		return
	}
}
