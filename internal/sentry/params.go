package sentry

import (
	_ "embed"
	"os"
)

//go:generate sh -c "echo $SENTRY_CONTEXT_NAME > SENTRY_CONTEXT_NAME"
//go:generate sh -c "echo $SENTRY_CONTEXT_VERSION > SENTRY_CONTEXT_VERSION"
//go:generate sh -c "echo $SENTRY_PRODUCTION > SENTRY_PRODUCTION"

const productionEnvironment = "production"

var (
	//go:embed SENTRY_CONTEXT_NAME
	defaultContextName string

	//go:embed SENTRY_CONTEXT_VERSION
	defaultContextVersion string

	//go:embed SENTRY_PRODUCTION
	production string
)

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

func environment(production bool, envvar string) string {
	if production {
		return productionEnvironment
	}
	env := os.Getenv(envvar)
	if env == productionEnvironment {
		// Production environment can only be set during build
		return ""
	}
	return env
}
