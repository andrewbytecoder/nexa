package psutil

import (
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	tAll            = "all"
	tUsage          = "usage"
	tIOCounter      = "IOCounter"
	tTimes          = "times"
	tPercent        = "percent"
	tInfo           = "info"
	tMem            = "mem"
	tSwap           = "swap"
	tSwapDev        = "swapDev"
	tUserStat       = "userStat"
	tLoadAvg        = "loadAvg"
	tLoadMisc       = "loadMisc"
	tNetIOCounter   = "netIOCounter"
	tNetIfConfig    = "netIfConfig"
	tNetConnections = "netConnections"
	tNetConntrack   = "netConntrack"
	tNetPids        = "netPids"
)

type PsUtil struct {
	ctx    *ctx.Ctx
	logger *zap.Logger

	psMem  *PsMem // 内存信息
	psCpu  *PsCpu
	psDisk *PsDisk
	psHost *PsHost
	psLoad *PsLoad
	psNet  *PsNet
}

func NewPsutil(ctx *ctx.Ctx) *PsUtil {
	psUtil := &PsUtil{
		ctx:    ctx,
		logger: ctx.Logger(),
	}
	// 创建内存信息对象
	psUtil.psMem = NewPsMem(psUtil)
	psUtil.psCpu = NewPsCpu(psUtil)
	psUtil.psDisk = NewPsDisk(psUtil)
	psUtil.psHost = NewPsHost(psUtil)
	psUtil.psLoad = NewPsLoad(psUtil)
	psUtil.psNet = NewPsnet(psUtil)

	return psUtil
}

func (psUtil *PsUtil) GetMemoryHandler() *PsMem {
	return psUtil.psMem
}

func (psUtil *PsUtil) GetCpuHandler() *PsCpu {
	return psUtil.psCpu
}

func (psUtil *PsUtil) GetDiskHandler() *PsDisk {
	return psUtil.psDisk
}
func (psUtil *PsUtil) GetHostHandler() *PsHost {
	return psUtil.psHost
}

func (psUtil *PsUtil) GetLoadHandler() *PsLoad {
	return psUtil.psLoad
}
func (psUtil *PsUtil) GetNetHandler() *PsNet {
	return psUtil.psNet
}

func (psUtil *PsUtil) ParseFlags(cmd *cobra.Command) {

	//cmd.Flags().StringVarP(&psUtil.httpMethod, "request", "X", "GET", "HTTP method to use")
	//cmd.Flags().StringVarP(&psUtil.postBody, "body", "d", "", "the body of a POST or PUT request; from file use @filename")
	//cmd.Flags().BoolVarP(&psUtil.followRedirects, "redirects", "L", false, "follow 30x redirects")
	//cmd.Flags().BoolVarP(&psUtil.onlyHeader, "readRequest", "I", false, "don't read body of request")
	//cmd.Flags().BoolVarP(&psUtil.insecure, "ssl", "k", false, "allow insecure SSL connections")
	//cmd.Flags().BoolVarP(&psUtil.saveOutput, "output", "O", false, "save body as remote filename")
	//cmd.Flags().StringVarP(&psUtil.outputFile, "save", "o", "", "output file for body")
	//cmd.Flags().StringVarP(&psUtil.clientCertFile, "cert", "E", "", "client cert file for tls config")
	//cmd.Flags().BoolVarP(&psUtil.fourOnly, "ipv4", "4", false, "resolve IPv4 addresses only")
	//cmd.Flags().BoolVarP(&psUtil.sixOnly, "ipv6", "6", false, "resolve IPv6 addresses only")
	//// 获取slice参数
	//cmd.Flags().VarP(&psUtil.httpHeaders, "header", "H", "set HTTP header; repeatable: -H 'Accept: ...' -H 'Range: ...'")
}
