package logutils

import (
	"context"
	"log/slog"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/log"
)

type gethAdapter struct {
	log.Logger
	target *zap.Logger
}

func newGethAdapter(target *zap.Logger) *gethAdapter {
	return &gethAdapter{
		Logger: log.NewLogger(log.DiscardHandler()),
		target: target,
	}
}

func (g *gethAdapter) serializeLog(level slog.Level, msg string, ctx ...interface{}) string {
	buf := strings.Builder{}
	logger := log.NewLogger(log.LogfmtHandlerWithLevel(&buf, level))
	logger.Log(level, msg, ctx...)
	return buf.String()
}

func (g *gethAdapter) Log(level slog.Level, msg string, ctx ...interface{}) {
	g.Write(level, msg, ctx...)
}

func (g *gethAdapter) Trace(msg string, ctx ...interface{}) {
	g.Write(log.LevelTrace, msg, ctx...)
}

func (g *gethAdapter) Debug(msg string, ctx ...interface{}) {
	g.Write(log.LevelDebug, msg, ctx...)
}

func (g *gethAdapter) Info(msg string, ctx ...interface{}) {
	g.Write(log.LevelInfo, msg, ctx...)
}

func (g *gethAdapter) Warn(msg string, ctx ...interface{}) {
	g.Write(log.LevelWarn, msg, ctx...)
}

func (g *gethAdapter) Error(msg string, ctx ...interface{}) {
	g.Write(log.LevelError, msg, ctx...)
}

func (g *gethAdapter) Crit(msg string, ctx ...interface{}) {
	g.Write(log.LevelCrit, msg, ctx...)
}

func (g *gethAdapter) Write(level slog.Level, msg string, attrs ...any) {
	g.target.Check(zapLevelFromGeth(level), g.serializeLog(level, msg, attrs...)).Write()
}

func (g *gethAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return g.target.Core().Enabled(zapLevelFromGeth(level)) && g.Logger.Enabled(ctx, level)
}

const traceLevel = zapcore.DebugLevel - 1

func zapLevelFromGeth(lvl slog.Level) zapcore.Level {
	switch lvl {
	case log.LevelTrace:
		return traceLevel // zap does not have a trace level, use custom
	case log.LevelDebug:
		return zapcore.DebugLevel
	case log.LevelInfo:
		return zapcore.InfoLevel
	case log.LevelWarn:
		return zapcore.WarnLevel
	case log.LevelError:
		return zapcore.ErrorLevel
	case log.LevelCrit:
		return zapcore.DPanicLevel // zap does not have a crit level, using DPanicLevel as closest
	default:
		return zapcore.InvalidLevel
	}
}
