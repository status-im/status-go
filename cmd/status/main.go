package main

import (
	"fmt"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/params"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
)

func main() {
	netVersion := "mainnet"
	if geth.UseTestnet {
		netVersion = "testnet"
	}
	fmt.Printf("%s\nVersion: %s\nGit Commit: %s\nBuild Date: %s\nNetwork: %s\n",
		geth.ClientIdentifier, params.Version, gitCommit, buildStamp, netVersion)
}
