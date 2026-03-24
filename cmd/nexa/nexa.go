package main

import (
	"fmt"

	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/register/cmdregister"
	"github.com/nexa/pkg/register/gops"
	"github.com/nexa/pkg/register/net"
	"github.com/nexa/pkg/register/psutil"
	"github.com/nexa/pkg/register/udp"
	"github.com/nexa/pkg/register/version"
)

func main() {

	cmdRegister := cmdregister.NewNexaCommand()
	cCtx := ctx.New()
	cmdRegister.
		// 注册http请求命令
		AddCommand(net.Cmd(cCtx)).
		AddCommand(psutil.GetPsUtilCmd(cCtx)).
		AddCommand(gops.GetGoPsCmd(cCtx)).
		AddCommand(version.GetVersionCmd(cCtx)).
		AddCommand(udp.GetCmd(cCtx))
	if err := cmdRegister.Execute(); err != nil {
		fmt.Println(err)
	}
}
