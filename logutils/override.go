package logutils

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/params"
)

// OverrideRootLogWithConfig derives all configuration from params.NodeConfig and configures logger using it.
func OverrideRootLogWithConfig(config *params.NodeConfig, colors bool) error {
	return OverrideRootLog(config.LogEnabled, config.LogLevel, FileOptions{
		Filename:   config.LogFile,
		MaxSize:    config.LogMaxSize,
		MaxBackups: config.LogMaxBackups,
		Compress:   config.LogCompressRotated,
	}, colors)

}

// OverrideRootLog overrides root logger with file handler, if defined,
// and log level (defaults to INFO).
func OverrideRootLog(enabled bool, levelStr string, fileOpts FileOptions, terminal bool) error {
	if !enabled {
		disableRootLog()
		return nil
	}

	return enableRootLog(levelStr, fileOpts, terminal)
}

func disableRootLog() {
	log.Root().SetHandler(log.DiscardHandler())
}

func enableRootLog(levelStr string, fileOpts FileOptions, terminal bool) error {
	var (
		handler log.Handler
		err     error
	)
	if fileOpts.Filename != "" {
		handler = FileHandlerWithRotation(fileOpts, log.LogfmtFormat())
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
