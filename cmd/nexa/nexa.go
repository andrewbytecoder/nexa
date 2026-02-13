package main

import (
	"fmt"

	"github.com/nexa/pkg/cmd/cmdregister"
	"github.com/nexa/pkg/cmd/gops"
	"github.com/nexa/pkg/cmd/httpstat"
	"github.com/nexa/pkg/cmd/psutil"
	"github.com/nexa/pkg/cmd/version"
	"github.com/nexa/pkg/ctx"
)

func main() {

	cmdRegister := cmdregister.NewNexaCommand()
	cCtx := ctx.New()
	cmdRegister.
		// 注册http请求命令
		AddCommand(httpstat.GetHttpCmd(cCtx)).
		AddCommand(psutil.GetPsUtilCmd(cCtx)).
		AddCommand(gops.GetGoPsCmd(cCtx)).
		AddCommand(version.GetVersionCmd(cCtx))
	if err := cmdRegister.Execute(); err != nil {
		fmt.Println(err)
	}
}
