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
// Link-time, not generated into files read back with go:embed: generating them
// writes into the source tree, and a consumer resolving status-go as a nimble
// dependency builds a read-only copy of this tree in place. The empty defaults
// mean no Sentry context and a non-production environment.
var (
	defaultContextName string

	defaultContextVersion string

	// production is "true" or "1" for a production build.
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
