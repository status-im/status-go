package sentry

import (
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/status-im/status-go/internal/version"
)

func Init(opts ...Option) error {
	cfg := defaultConfig()
	applyOptions(cfg, opts...)
	return sentry.Init(*cfg)
}

func MustInit(options ...Option) {
	if err := Init(options...); err != nil {
		panic(err)
	}
}

func Close() error {
	sentry.Flush(time.Second * 2)
	// Set DSN to empty string to disable sending events
	return sentry.Init(sentry.ClientOptions{
		Dsn: "",
	})
}

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	RecoverError(err)
	panic(err)
}

func RecoverError(err interface{}) {
	sentry.CurrentHub().Recover(err)
	sentry.Flush(time.Second * 2)
}

func defaultConfig() *sentry.ClientOptions {
	return &sentry.ClientOptions{
		EnableTracing:  false,
		Debug:          false,
		SendDefaultPII: false,
		Release:        version.Version(),
		Environment:    Environment(),
		Tags:           make(map[string]string),
		BeforeSend:     beforeSend,
	}
}

func beforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	event.Modules = nil   // Clear modules as we know all dependencies by commit hash
	event.ServerName = "" // Clear server name as it might be sensitive
	return event
}
