package logutils

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// LvlFromString returns the appropriate zapcore.Level from a string.
func LvlFromString(lvlString string) (zapcore.Level, error) {
	switch strings.ToLower(lvlString) {
	case "trace", "trce":
		return traceLevel, nil // zap does not have a trace level, use custom
	case "debug", "dbug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error", "eror":
		return zapcore.ErrorLevel, nil
	case "crit":
		return zapcore.DPanicLevel, nil // zap does not have a crit level, using DPanicLevel as closest
	default:
		return zapcore.InvalidLevel, fmt.Errorf("unknown level: %v", lvlString)
	}
}
