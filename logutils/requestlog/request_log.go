package requestlog

import (
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/zaputil"
)

var (
	requestLogger *zap.Logger
)

// IsRequestLoggingEnabled returns whether RPC logging is enabled
func IsRequestLoggingEnabled() bool {
	return requestLogger != nil
}

// GetRequestLogger returns the RPC logger object
func GetRequestLogger() *zap.Logger {
	return requestLogger
}

func ConfigureAndEnableRequestLogging(file string) error {
	if len(file) == 0 {
		return errors.New("file is required")
	}

	if IsRequestLoggingEnabled() {
		return errors.New("request logging is already enabled")
	}

	fileOpts := logutils.FileOptions{
		Filename:   file,
		MaxBackups: 1,
	}

	core := zapcore.NewCore(
		zaputil.NewConsoleHexEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(logutils.ZapSyncerWithRotation(fileOpts)),
		zap.DebugLevel,
	)

	requestLogger = zap.New(core).Named("RequestLogger")

	return nil
}
