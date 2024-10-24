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

// GetRequestLogger returns the RPC logger object
func GetRequestLogger() *zap.Logger {
	return requestLogger
}

func CreateRequestLogger(file string) (*zap.Logger, error) {
	if len(file) == 0 {
		return nil, errors.New("file is required")
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

	return zap.New(core).Named("RequestLogger"), nil
}

func ConfigureAndEnableRequestLogging(file string) error {
	logger, err := CreateRequestLogger(file)
	if err != nil {
		return err
	}

	requestLogger = logger

	return nil
}
