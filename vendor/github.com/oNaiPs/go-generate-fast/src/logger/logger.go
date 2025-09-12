package logger

import (
	"github.com/oNaiPs/go-generate-fast/src/core/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init() {
	var zapConfig zap.Config

	isDevelopment := false
	isDebug := false
	if config.Get() != nil {
		isDebug = config.Get().Debug
	}

	if isDevelopment {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	zapConfig.Encoding = "console"
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapConfig.EncoderConfig.TimeKey = zapcore.OmitKey

	if !isDevelopment {
		if !isDebug {
			zapConfig.EncoderConfig.CallerKey = ""
			zapConfig.EncoderConfig.StacktraceKey = ""
			zapConfig.EncoderConfig.LevelKey = ""
			zapConfig.Level.SetLevel(zapcore.InfoLevel)
		} else {
			zapConfig.Level.SetLevel(zapcore.DebugLevel)
		}
	}

	logger, _ := zapConfig.Build()

	zap.ReplaceGlobals(logger)
}
