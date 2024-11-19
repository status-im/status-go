package sentry

import (
	"os"

	"github.com/getsentry/sentry-go"
)

const DefaultEnvVarDSN = "SENTRY_DSN_STATUS_GO"
const DefaultEnvVarEnvironment = "SENTRY_ENVIRONMENT"

type Option func(*sentry.ClientOptions)

func WithDSN(dsn string) Option {
	return func(o *sentry.ClientOptions) {
		o.Dsn = dsn
	}
}

func WithEnvironmentDSN(name string) Option {
	return WithDSN(os.Getenv(name))
}

//func WithEnvironmentDSN() Option {
//	return WithDSN(os.Getenv("SENTRY_DSN"))
//}

func WithContext(name string, version string) Option {
	return func(o *sentry.ClientOptions) {
		if o.Tags == nil {
			o.Tags = make(map[string]string)
		}
		o.Tags["context.name"] = name
		o.Tags["context.version"] = version
	}
}

func WithDefaultContext() Option {
	return WithContext(DefaultContext(), DefaultContextVersion())
}

func applyOptions(cfg *sentry.ClientOptions, opts ...Option) {
	for _, opt := range opts {
		opt(cfg)
	}
}
