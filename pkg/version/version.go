package version

// Set at link time by the Makefile (BUILD_VARS_LDFLAGS), so building never
// writes into the source tree. These defaults are what a plain `go build` gets.
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
