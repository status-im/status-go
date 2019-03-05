package logutils

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/params"
)

// OverrideWithStdLogger overwrites ethereum's root logger with a logger from golang std lib.
func OverrideWithStdLogger(config *params.NodeConfig) error {
	return enableRootLog(config.LogLevel, NewStdHandler(log.LogfmtFormat()))
}

// OverrideRootLogWithConfig derives all configuration from params.NodeConfig and configures logger using it.
func OverrideRootLogWithConfig(config *params.NodeConfig, colors bool) error {
	if !config.LogEnabled {
		return nil
	}
	if config.LogMobileSystem {
		return OverrideWithStdLogger(config)
	}
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
	var (
		handler log.Handler
	)
	if fileOpts.Filename != "" {
		handler = FileHandlerWithRotation(fileOpts, log.LogfmtFormat())
	} else {
		handler = log.StreamHandler(os.Stderr, log.TerminalFormat(terminal))
	}

	return enableRootLog(levelStr, handler)
}

func disableRootLog() {
	log.Root().SetHandler(log.DiscardHandler())
}

func enableRootLog(levelStr string, handler log.Handler) error {
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
