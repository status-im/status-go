package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	versionCommand = cli.Command{
		Action: versionCommandHandler,
		Name:   "version",
		Usage:  "Print app version",
	}
)

// versionCommandHandler displays app version
func versionCommandHandler(ctx *cli.Context) error {
	fmt.Println(strings.Title(params.ClientIdentifier))
	fmt.Println("Version:", params.Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if buildStamp != "" {
		fmt.Println("Build Stamp:", buildStamp)
	}

	fmt.Println("Network Id:", ctx.GlobalInt(NetworkIDFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	printNodeConfig(ctx)

	return nil
}
