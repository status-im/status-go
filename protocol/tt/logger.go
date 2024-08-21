package tt

import (
	"github.com/status-im/status-go/protocol/zaputil"

	"go.uber.org/zap"
)

// MustCreateTestLogger returns a logger based on the passed flags.
func MustCreateTestLogger() *zap.Logger {
	return MustCreateTestLoggerWithConfig(loggerConfig())
}

func MustCreateTestLoggerWithConfig(cfg zap.Config) *zap.Logger {
	if err := zaputil.RegisterConsoleHexEncoder(); err != nil {
		panic(err)
	}
	cfg.Encoding = "console-hex"
	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return l
}
