package gops

import (
	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type PsCmd struct {
	ctx    *ctx.Ctx
	logger *zap.Logger
}

func NewGoPs(ctx *ctx.Ctx) *PsCmd {
	psUtil := &PsCmd{
		ctx:    ctx,
		logger: ctx.Logger(),
	}
	// 创建内存信息对象

	return psUtil
}

func (psUtil *PsCmd) ParseFlags(cmd *cobra.Command) {

	//register.Flags().StringVarP(&psUtil.httpMethod, "request", "X", "GET", "HTTP method to use")
	//register.Flags().StringVarP(&psUtil.postBody, "body", "d", "", "the body of a POST or PUT request; from file use @filename")
	//register.Flags().BoolVarP(&psUtil.followRedirects, "redirects", "L", false, "follow 30x redirects")
	//register.Flags().BoolVarP(&psUtil.onlyHeader, "readRequest", "I", false, "don't read body of request")
	//register.Flags().BoolVarP(&psUtil.insecure, "ssl", "k", false, "allow insecure SSL connections")
	//register.Flags().BoolVarP(&psUtil.saveOutput, "output", "O", false, "save body as remote filename")
	//register.Flags().StringVarP(&psUtil.outputFile, "save", "o", "", "output file for body")
	//register.Flags().StringVarP(&psUtil.clientCertFile, "cert", "E", "", "client cert file for tls config")
	//register.Flags().BoolVarP(&psUtil.fourOnly, "ipv4", "4", false, "resolve IPv4 addresses only")
	//register.Flags().BoolVarP(&psUtil.sixOnly, "ipv6", "6", false, "resolve IPv6 addresses only")
	//// 获取slice参数
	//register.Flags().VarP(&psUtil.httpHeaders, "header", "H", "set HTTP header; repeatable: -H 'Accept: ...' -H 'Range: ...'")
}
