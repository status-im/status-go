package main

import (
	"fmt"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"
)

func main() {
	fmt.Printf("Status\nGit Commit: %s\nBuild Time: %s\n", gitCommit, buildStamp)
}
