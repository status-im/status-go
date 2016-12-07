package main

import (
	"fmt"
	"github.com/status-im/status-go/geth"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
)

func main() {
	verString := fmt.Sprintf("%d.%d.%d", geth.VersionMajor, geth.VersionMinor, geth.VersionPatch)
	if geth.VersionMeta != "" {
		verString += "-" + geth.VersionMeta
	}
	if gitCommit != "" {
		verString += "-" + gitCommit[:8]
	}
	netVersion := "mainnet"
	if geth.UseTestnet {
		netVersion = "testnet"
	}
	fmt.Printf("Status\nGit Commit: %s\nBuild Time: %s\nVersion: %s\nNetwork: %s\n",
		gitCommit, buildStamp, verString, netVersion)
}
