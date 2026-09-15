package sentry

import (
	"os"
)

// defaultContextName, defaultContextVersion and production are set at LINK
// time by the Makefile from the SENTRY_CONTEXT_NAME, SENTRY_CONTEXT_VERSION
// and SENTRY_PRODUCTION environment variables:
//
//	-ldflags "-X github.com/status-im/status-go/pkg/sentry.defaultContextName=..."
//
// They used to be `go:generate sh -c "echo $VAR > FILE"` outputs read back
// with go:embed, which made a build write into the source tree — impossible
// for a consumer that builds the read-only nimble store copy of this tree in
// place. Same env var names, same meaning; only the transport changed. The
// empty defaults are what a plain `go build` gets: no Sentry context and a
// non-production environment.
var (
	defaultContextName string

	defaultContextVersion string

	// production is "true"/"1" when the build is a production build.
	production string
)

const productionEnvironment = "production"

func DefaultContext() string {
	return defaultContextName
}

func DefaultContextVersion() string {
	return defaultContextVersion
}

func Production() bool {
	return production == "true" || production == "1"
}

func Environment() string {
	return environment(Production(), DefaultEnvVarEnvironment)
}

func environment(forceProduction bool, envvar string) string {
	if forceProduction {
		return productionEnvironment
	}
	env := os.Getenv(envvar)
	if env == productionEnvironment {
		// Production environment can only be set during build
		return ""
	}
	return env
}
