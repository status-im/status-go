package logutils

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/log"
)

// gethAdapter returns a log.Handler interface that forwards logs to a zap.Logger.
// Logs are forwarded raw as if geth were printing them.
func gethAdapter(logger *zap.Logger) log.Handler {
	return log.FuncHandler(func(r *log.Record) error {
		level, err := lvlFromString(r.Lvl.String())
		if err != nil {
			return err
		}
		// Skip trace logs to not clutter the output
		if level == traceLevel {
			return nil
		}
		serializedLog := string(log.TerminalFormat(false).Format(r))
		logger.Check(level, fmt.Sprintf("'%s'", strings.TrimSuffix(serializedLog, "\n"))).Write()
		return nil
	})
}

const traceLevel = zapcore.DebugLevel - 1

// lvlFromString returns the appropriate zapcore.Level from a string.
func lvlFromString(lvlString string) (zapcore.Level, error) {
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
