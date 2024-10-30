package params

import _ "embed"

//go:generate sh -c "../_assets/scripts/version.sh > VERSION"
//go:generate sh -c "git rev-parse --short HEAD > GIT_COMMIT"

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

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}
