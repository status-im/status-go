package version

// version and gitCommit are set at LINK time by the Makefile:
//
//	-ldflags "-X github.com/status-im/status-go/pkg/version.version=... \
//	          -X github.com/status-im/status-go/pkg/version.gitCommit=..."
//
// Link-time, not generated into files read back with go:embed: generating them
// writes into the source tree, and a consumer resolving status-go as a nimble
// dependency builds a read-only copy of this tree in place. The Makefile takes
// the values from `git describe --tags` and `git rev-parse --short HEAD`; the
// defaults below are what a plain `go build` with no -ldflags gets.
var (
	version = "0.0.0-dev"

	gitCommit = "unknown"
)

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}
