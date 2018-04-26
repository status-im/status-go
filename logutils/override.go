package logutils

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

// OverrideRootLog overrides root logger with file handler, if defined,
// and log level (defaults to INFO).
func OverrideRootLog(enabled bool, levelStr string, logFile string, terminal bool) error {
	if !enabled {
		disableRootLog()
		return nil
	}

	return enableRootLog(levelStr, logFile, terminal)
}

func disableRootLog() {
	log.Root().SetHandler(log.DiscardHandler())
}

func enableRootLog(levelStr string, logFile string, terminal bool) error {
	var (
		handler log.Handler
		err     error
	)

	if logFile != "" {
		handler, err = log.FileHandler(logFile, log.LogfmtFormat())
		if err != nil {
			return err
		}
	} else {
		handler = log.StreamHandler(os.Stderr, log.TerminalFormat(terminal))
	}

	if levelStr == "" {
		levelStr = "INFO"
	}

	level, err := log.LvlFromString(strings.ToLower(levelStr))
	if err != nil {
		return err
	}

	filteredHandler := log.LvlFilterHandler(level, handler)
	log.Root().SetHandler(filteredHandler)

	return nil
}
