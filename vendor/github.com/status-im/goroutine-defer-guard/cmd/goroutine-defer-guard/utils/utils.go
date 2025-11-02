package utils

import (
	"strconv"

	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func URI(path string, line int) string {
	return path + ":" + strconv.Itoa(line)
}

func ZapURI(path string, line int) zap.Field {
	return zap.Field{
		Type:   zapcore.StringType,
		Key:    "uri",
		String: URI(path, line),
	}
}

func BuildLogger(level zapcore.Level) *zap.Logger {
	// Initialize logger with colors
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(level)
	loggerConfig.Development = false
	loggerConfig.DisableStacktrace = true
	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Printf("failed to initialize logger: %s", err.Error())
		os.Exit(1)
	}

	return logger.Named("main")
}
