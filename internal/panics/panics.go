// Package panics carries the goroutine panic guard.
//
// Kept out of internal/logutils so that the Sentry SDK stays out of the
// dependency graph of a package almost everything imports.
package panics

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/sentry"
)

// LogOnPanic logs a panicking goroutine, reports it to Sentry and re-panics.
// Every spawned goroutine must `defer panics.LogOnPanic()`; `make lint-panics`
// enforces it.
func LogOnPanic() {
	err := recover()
	if err == nil {
		return
	}

	logutils.ZapLogger().Error("panic in goroutine",
		zap.Any("error", err),
		zap.Stack("stacktrace"))

	sentry.RecoverError(err)

	panic(err)
}
