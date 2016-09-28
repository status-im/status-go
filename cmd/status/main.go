package main

import (
	"fmt"
)

var (
	gitCommit  = "rely on linker: -ldflags -X main.GitCommit"
	buildStamp = "rely on linker: -ldflags -X main.buildStamp"

	versionMajor = 0          // Major version component of the current release
	versionMinor = 9          // Minor version component of the current release
	versionPatch = 1          // Patch version component of the current release
	versionMeta  = "unstable" // Version metadata to append to the version string
)

func main() {
	verString := fmt.Sprintf("%d.%d.%d", versionMajor, versionMinor, versionPatch)
	if versionMeta != "" {
		verString += "-" + versionMeta
	}
	if gitCommit != "" {
		verString += "-" + gitCommit[:8]
	}
	fmt.Printf("Status\nGit Commit: %s\nBuild Time: %s\nVersion: %s\n", gitCommit, buildStamp, verString)
}
