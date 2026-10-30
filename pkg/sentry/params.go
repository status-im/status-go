package sentry

import (
	"os"
)

// Set at link time by the Makefile (BUILD_VARS_LDFLAGS) from SENTRY_CONTEXT_NAME,
// SENTRY_CONTEXT_VERSION and SENTRY_PRODUCTION. Empty means no Sentry context
// and a non-production environment.
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
