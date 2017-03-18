package main

import (
	"fmt"
	"github.com/status-im/status-go/geth/params"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
)

func main() {
	nodeConfig, err := params.NewNodeConfig(".ethereumcmd", params.TestNetworkId)
	if err != nil {
		panic(err)
	}

	netVersion := "mainnet"
	if nodeConfig.TestNet {
		netVersion = "testnet"
	}

	fmt.Printf("%s\nVersion: %s\nGit Commit: %s\nBuild Date: %s\nNetwork: %s\n",
		nodeConfig.Name, params.Version, gitCommit, buildStamp, netVersion)
}
