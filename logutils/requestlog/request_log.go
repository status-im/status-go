package requestlog

import (
	"errors"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/logutils"
)

var (
	// requestLogger is the request logger object
	requestLogger log.Logger
	// isRequestLoggingEnabled controls whether request logging is enabled
	isRequestLoggingEnabled atomic.Bool
)

// NewRequestLogger creates a new request logger object
func NewRequestLogger(ctx ...interface{}) log.Logger {
	requestLogger = log.New(ctx...)
	return requestLogger
}

// EnableRequestLogging enables or disables RPC logging
func EnableRequestLogging(enable bool) {
	if enable {
		isRequestLoggingEnabled.Store(true)
	} else {
		isRequestLoggingEnabled.Store(false)
	}
}

// IsRequestLoggingEnabled returns whether RPC logging is enabled
func IsRequestLoggingEnabled() bool {
	return isRequestLoggingEnabled.Load()
}

// GetRequestLogger returns the RPC logger object
func GetRequestLogger() log.Logger {
	return requestLogger
}

func ConfigureAndEnableRequestLogging(file string) error {
	log.Info("initialising request logger", "log file", file)
	requestLogger := NewRequestLogger()
	if file == "" {
		return errors.New("log file path is required")
	}
	fileOpts := logutils.FileOptions{
		Filename:   file,
		MaxBackups: 1,
	}
	handler := logutils.FileHandlerWithRotation(fileOpts, log.LogfmtFormat())
	filteredHandler := log.LvlFilterHandler(log.LvlDebug, handler)
	requestLogger.SetHandler(filteredHandler)
	EnableRequestLogging(true)
	return nil
}
