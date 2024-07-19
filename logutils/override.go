package logutils

import (
	"fmt"
	stdlog "log"
	"log/slog"
	"os"
	"strings"

	logging "github.com/ipfs/go-log/v2"

	"github.com/ethereum/go-ethereum/log"
)

type LogSettings struct {
	Enabled         bool
	MobileSystem    bool
	Level           string
	File            string
	MaxSize         int
	MaxBackups      int
	CompressRotated bool
}

// OverrideWithStdLogger overwrites ethereum's root logger with a logger from golang std lib.
func OverrideWithStdLogger(logLevel string) error {
	level, err := lvlFromString(logLevel)
	if err != nil {
		return err
	}
	return enableRootLog(logLevel, log.NewTerminalHandlerWithLevel(stdlog.Writer(), level, false))
}

// OverrideRootLogWithConfig derives all configuration from params.NodeConfig and configures logger using it.
func OverrideRootLogWithConfig(settings LogSettings, colors bool) error {
	if !settings.Enabled {
		return nil
	}
	if settings.MobileSystem {
		return OverrideWithStdLogger(settings.Level)
	}
	return OverrideRootLog(settings.Enabled, settings.Level, FileOptions{
		Filename:   settings.File,
		MaxSize:    settings.MaxSize,
		MaxBackups: settings.MaxBackups,
		Compress:   settings.CompressRotated,
	}, colors)

}

// OverrideRootLog overrides root logger with file handler, if defined,
// and log level (defaults to INFO).
func OverrideRootLog(enabled bool, levelStr string, fileOpts FileOptions, terminal bool) error {
	if !enabled {
		disableRootLog()
		return nil
	}
	if os.Getenv("CI") == "true" {
		terminal = false
	}
	var (
		handler slog.Handler
	)

	level, err := lvlFromString(levelStr)

	if err != nil {
		return err
	}
	if fileOpts.Filename != "" {
		if fileOpts.MaxBackups == 0 {
			// Setting MaxBackups to 0 causes all log files to be kept. Even setting MaxAge to > 0 doesn't fix it
			// Docs: https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2@v2.0.0#readme-cleaning-up-old-log-files
			fileOpts.MaxBackups = 1
		}
		handler = FileHandlerWithRotation(fileOpts, level, terminal)
	} else {
		handler = log.NewTerminalHandlerWithLevel(os.Stderr, level, terminal)
	}

	return enableRootLog(levelStr, handler)
}

func disableRootLog() {
	log.SetDefault(log.NewLogger(log.DiscardHandler()))
}

// LvlFromString returns the appropriate slog.Level from a string name.
// Useful for parsing command line args and configuration files.
func lvlFromString(lvlString string) (slog.Level, error) {
	switch lvlString {
	case "trace", "trce":
		return log.LevelTrace, nil
	case "debug", "dbug":
		return log.LevelDebug, nil
	case "info":
		return log.LevelInfo, nil
	case "warn":
		return log.LevelWarn, nil
	case "error", "eror":
		return log.LevelError, nil
	case "crit":
		return log.LevelCrit, nil
	default:
		return log.LevelDebug, fmt.Errorf("unknown level: %v", lvlString)
	}
}

func enableRootLog(levelStr string, handler slog.Handler) error {
	log.SetDefault(log.NewLogger(handler))

	// go-libp2p logger
	if levelStr == "" {
		levelStr = "INFO"
	}
	levelStr = strings.ToLower(levelStr)
	lvl, err := logging.LevelFromString(levelStr)
	if err != nil {
		return err
	}

	logging.SetAllLoggers(lvl)

	return nil
}
