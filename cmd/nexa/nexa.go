package main

import (
	"fmt"

	"github.com/nexa/cmd/nexa/gops"
	"github.com/nexa/cmd/nexa/helmify"
	"github.com/nexa/cmd/nexa/kube"
	"github.com/nexa/cmd/nexa/net"
	"github.com/nexa/cmd/nexa/node"
	"github.com/nexa/cmd/nexa/prometheus"
	"github.com/nexa/cmd/nexa/psutil"
	"github.com/nexa/cmd/nexa/udp"
	"github.com/nexa/cmd/nexa/version"
	"github.com/nexa/pkg/ctx"
)

func main() {

	cmdRegister := NewNexaCommand()
	cCtx := ctx.New()
	cmdRegister.
		AddCommand(net.Cmd(cCtx)).
		AddCommand(node.Cmd(cCtx)).
		AddCommand(psutil.GetPsUtilCmd(cCtx)).
		AddCommand(gops.GetGoPsCmd(cCtx)).
		AddCommand(version.GetVersionCmd(cCtx)).
		AddCommand(udp.GetCmd(cCtx)).
		AddCommand(helmify.GetCmd(cCtx)).
		AddCommand(prometheus.Cmd(cCtx)).
		AddCommand(kube.GetCmd(cCtx))
	if err := cmdRegister.Execute(); err != nil {
		fmt.Println(err)
	}
}
