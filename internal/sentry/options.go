package sentry

import (
	"os"

	"github.com/getsentry/sentry-go"
)

type Option func(*sentry.ClientOptions)

func WithDSN(dsn string) Option {
	return func(o *sentry.ClientOptions) {
		o.Dsn = dsn
	}
}

func WithEnvironmentDSN() Option {
	return WithDSN(os.Getenv("SENTRY_DSN"))
}

func WithContext(name string, version string) Option {
	return func(o *sentry.ClientOptions) {
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
