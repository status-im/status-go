package logutils

import (
	"github.com/ethereum/go-ethereum/log"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// FileOptions are all options supported by internal rotation module.
type FileOptions struct {
	// Base name for log file.
	Filename string
	// Size in megabytes.
	MaxSize int
	// Number of rotated log files.
	MaxBackups int
	// If true rotated log files will be gzipped.
	Compress bool
}

// FileHandlerWithRotation instantiates log.Handler with a configured rotation
func FileHandlerWithRotation(opts FileOptions, format log.Format) log.Handler {
	logger := &lumberjack.Logger{
		Filename:   opts.Filename,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		Compress:   opts.Compress,
	}
	return log.StreamHandler(logger, format)
}
