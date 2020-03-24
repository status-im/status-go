package tt

import (
	"os"
	"sync"

	"github.com/status-im/status-go/protocol/zaputil"

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
	l := zap.NewNop()
	for _, arg := range os.Args {
		if arg == "-v" || arg == "-test.v" {
			cfg := zap.NewDevelopmentConfig()
			cfg.Encoding = "console-hex"
			var err error
			l, err = cfg.Build()
			if err != nil {
				panic(err)
			}
		}
	}
	return l
}
