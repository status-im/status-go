package utils

import (
	stdlog "log"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
)

func SetupLogging(logLevel *string, logWithoutColors *bool, config *params.NodeConfig) {
	if *logLevel != "" {
		config.LogLevel = *logLevel
	}

	logSettings := config.LogSettings()
	logSettings.Colorized = !(*logWithoutColors) && terminal.IsTerminal(int(os.Stdin.Fd()))
	if err := logutils.OverrideRootLoggerWithConfig(logSettings); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}
}
