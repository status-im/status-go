package version

// version and gitCommit are set at LINK time by the Makefile:
//
//	-ldflags "-X github.com/status-im/status-go/pkg/version.version=... \
//	          -X github.com/status-im/status-go/pkg/version.gitCommit=..."
//
// They used to be `go:generate sh -c "git … > FILE"` outputs read back with
// go:embed. That made a build WRITE into the source tree, which a consumer
// resolving status-go as a nimble dependency cannot do: it builds the
// read-only store copy of this tree in place. The sources of the values are
// unchanged (`git describe --tags`, `git rev-parse --short HEAD`); only the
// transport is. The defaults below are what a plain `go build` with no
// -ldflags gets, i.e. a developer build.
var (
	// version is defined in git tags.
	version = "0.0.0-dev"

	// gitCommit is a commit hash.
	gitCommit = "unknown"
)

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}
