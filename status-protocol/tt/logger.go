package tt

import (
	"sync"

	"github.com/status-im/status-protocol-go/zaputil"

	"go.uber.org/zap"
)

var registerOnce sync.Once

// MustCreateTestLogger returns a logger based on the passed flags.
func MustCreateTestLogger() *zap.Logger {
	registerOnce.Do(func() {
		if err := zaputil.RegisterConsoleHexEncoder(); err != nil {
			panic(err)
		}
	})

	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console-hex"
	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return l
}
