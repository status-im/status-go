package logutils

import (
	tt "github.com/status-im/status-go/protocol/tt"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/log"
)

// Logger returns the main logger instance used by status-go.
func Logger() log.Logger {
	return log.Root()
}

var (
	_zapLogger     *zap.Logger
	_initZapLogger sync.Once
)

// ZapLogger creates a custom zap.Logger which will forward logs
// to status-go logger.
func ZapLogger() *zap.Logger {
	_initZapLogger.Do(func() {
		_zapLogger = tt.MustCreateTestLogger()
	})
	return _zapLogger
}
