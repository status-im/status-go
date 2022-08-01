package utils

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger = nil
var atom = zap.NewAtomicLevel()

// SetLogLevel sets a custom log level
func SetLogLevel(level string) error {
	lvl := zapcore.InfoLevel // zero value
	err := lvl.Set(level)
	if err != nil {
		return err
	}
	atom.SetLevel(lvl)
	return nil
}

// Logger creates a zap.Logger with some reasonable defaults
func Logger() *zap.Logger {
	if log == nil {
		InitLogger("console")
	}
	return log
}

// InitLogger initializes a global logger using an specific encoding
func InitLogger(encoding string) {
	cfg := zap.Config{
		Encoding:         encoding,
		Level:            atom,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			TimeKey:      "time",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			NameKey:      "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Errorf("could not create logger: %s", err.Error()))
	}

	log = logger.Named("gowaku")
}
