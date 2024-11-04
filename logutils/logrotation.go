package logutils

import (
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
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

// ZapSyncerWithRotation creates a zapcore.WriteSyncer with a configured rotation
func ZapSyncerWithRotation(opts FileOptions) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   opts.Filename,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		Compress:   opts.Compress,
	})
}
