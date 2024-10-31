package params

import (
	_ "embed"
	"strings"
)

// Use go:generate script to get the version and git commit.
// VERSION and GIT_COMMIT files are used in further `go:embed` commands to load values to the variables.
// Suppress errors, assuming files have already been properly generated. Required for Docker builds.
//go:generate sh -c "../_assets/scripts/version.sh > VERSION || true"
//go:generate sh -c "git rev-parse --short HEAD > GIT_COMMIT || true"

var (
	// version is defined in git tags.
	// We set it from the Makefile.
	//go:embed VERSION
	version string

	// gitCommit is a commit hash.
	//go:embed GIT_COMMIT
	gitCommit string
)

// IpfsGatewayURL is the Gateway URL to use for IPFS
const IpfsGatewayURL = "https://ipfs.status.im/"

func init() {
	version = strings.TrimSpace(version)
	gitCommit = strings.TrimSpace(gitCommit)
}

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}
