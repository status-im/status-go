package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger = nil
var atom = zap.NewAtomicLevel()

func SetLogLevel(level string) error {
	lvl := zapcore.InfoLevel // zero value
	err := lvl.Set(level)
	if err != nil {
		return err
	}
	atom.SetLevel(lvl)
	return nil
}

func Logger() *zap.Logger {
	if log == nil {
		cfg := zap.Config{
			Encoding:         "console",
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
			panic("could not create logger")
		}

		log = logger.Named("gowaku")
	}
	return log
}
