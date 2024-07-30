package log

import (
	"context"
	"log/slog"
	"math"
	"os"
	"runtime"
	"time"
)

const errorKey = "LOG_ERROR"

const (
	legacyLevelCrit = iota
	legacyLevelError
	legacyLevelWarn
	legacyLevelInfo
	legacyLevelDebug
	legacyLevelTrace
)

const (
	levelMaxVerbosity slog.Level = math.MinInt
	LevelTrace        slog.Level = -8
	LevelDebug                   = slog.LevelDebug
	LevelInfo                    = slog.LevelInfo
	LevelWarn                    = slog.LevelWarn
	LevelError                   = slog.LevelError
	LevelCrit         slog.Level = 12

	// for backward-compatibility
	LvlTrace = LevelTrace
	LvlInfo  = LevelInfo
	LvlDebug = LevelDebug
)

// convert from old Geth verbosity level constants
// to levels defined by slog
func FromLegacyLevel(lvl int) slog.Level {
	switch lvl {
	case legacyLevelCrit:
		return LevelCrit
	case legacyLevelError:
		return slog.LevelError
	case legacyLevelWarn:
		return slog.LevelWarn
	case legacyLevelInfo:
		return slog.LevelInfo
	case legacyLevelDebug:
		return slog.LevelDebug
	case legacyLevelTrace:
		return LevelTrace
	default:
		break
	}

	// TODO: should we allow use of custom levels or force them to match existing max/min if they fall outside the range as I am doing here?
	if lvl > legacyLevelTrace {
		return LevelTrace
	}
	return LevelCrit
}

// LevelAlignedString returns a 5-character string containing the name of a Lvl.
func LevelAlignedString(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO "
	case slog.LevelWarn:
		return "WARN "
	case slog.LevelError:
		return "ERROR"
	case LevelCrit:
		return "CRIT "
	default:
		return "unknown level"
	}
}

// LevelString returns a string containing the name of a Lvl.
func LevelString(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "trace"
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "error"
	case LevelCrit:
		return "crit"
	default:
		return "unknown"
	}
}

// A Logger writes key/value pairs to a Handler
type Logger interface {
	// With returns a new Logger that has this logger's attributes plus the given attributes
	With(ctx ...interface{}) Logger

	// With returns a new Logger that has this logger's attributes plus the given attributes. Identical to 'With'.
	New(ctx ...interface{}) Logger

	// Log logs a message at the specified level with context key/value pairs
	Log(level slog.Level, msg string, ctx ...interface{})

	// Trace log a message at the trace level with context key/value pairs
	Trace(msg string, ctx ...interface{})

	// Debug logs a message at the debug level with context key/value pairs
	Debug(msg string, ctx ...interface{})

	// Info logs a message at the info level with context key/value pairs
	Info(msg string, ctx ...interface{})

	// Warn logs a message at the warn level with context key/value pairs
	Warn(msg string, ctx ...interface{})

	// Error logs a message at the error level with context key/value pairs
	Error(msg string, ctx ...interface{})

	// Crit logs a message at the crit level with context key/value pairs, and exits
	Crit(msg string, ctx ...interface{})

	// Write logs a message at the specified level
	Write(level slog.Level, msg string, attrs ...any)

	// Enabled reports whether l emits log records at the given context and level.
	Enabled(ctx context.Context, level slog.Level) bool

	// Handler returns the underlying handler of the inner logger.
	Handler() slog.Handler
}

type LoggerOptions struct {
	// Number of callers to skip in case when
	// source code location is required
	// (useful e.g. when using wrappers like zap)
	SkipCallers int
	// Use UTC time when creating a slog.Record instance
	UseUTC bool
}

type logger struct {
	inner *slog.Logger
	opts  *LoggerOptions
}

// NewLogger returns a logger with the specified handler set
func NewLogger(h slog.Handler) Logger {
	return &logger{
		slog.New(h),
		nil,
	}
}

// NewLoggerWithOpts returns a logger
// with the specified handler and options set
func NewLoggerWithOpts(h slog.Handler, opts *LoggerOptions) Logger {
	return &logger{
		slog.New(h),
		opts,
	}
}

func (l *logger) Handler() slog.Handler {
	return l.inner.Handler()
}

// write logs a message at the specified level:
func (l *logger) Write(level slog.Level, msg string, attrs ...any) {
	if !l.inner.Enabled(context.Background(), level) {
		return
	}

	if len(attrs)%2 != 0 {
		attrs = append(attrs, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	curTime := time.Now()
	if l.opts != nil && l.opts.UseUTC {
		curTime = curTime.UTC()
	}
	var pcs [1]uintptr
	var skipCallers int
	if l.opts != nil {
		skipCallers = l.opts.SkipCallers
	}
	runtime.Callers(3+skipCallers, pcs[:])
	r := slog.NewRecord(curTime, level, msg, pcs[0])
	r.Add(attrs...)
	l.inner.Handler().Handle(context.Background(), r)
}

func (l *logger) Log(level slog.Level, msg string, attrs ...any) {
	l.Write(level, msg, attrs...)
}

func (l *logger) With(ctx ...interface{}) Logger {
	return &logger{l.inner.With(ctx...), l.opts}
}

func (l *logger) New(ctx ...interface{}) Logger {
	return l.With(ctx...)
}

// Enabled reports whether l emits log records at the given context and level.
func (l *logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.inner.Enabled(ctx, level)
}

func (l *logger) Trace(msg string, ctx ...interface{}) {
	l.Write(LevelTrace, msg, ctx...)
}

func (l *logger) Debug(msg string, ctx ...interface{}) {
	l.Write(slog.LevelDebug, msg, ctx...)
}

func (l *logger) Info(msg string, ctx ...interface{}) {
	l.Write(slog.LevelInfo, msg, ctx...)
}

func (l *logger) Warn(msg string, ctx ...any) {
	l.Write(slog.LevelWarn, msg, ctx...)
}

func (l *logger) Error(msg string, ctx ...interface{}) {
	l.Write(slog.LevelError, msg, ctx...)
}

func (l *logger) Crit(msg string, ctx ...interface{}) {
	l.Write(LevelCrit, msg, ctx...)
	os.Exit(1)
}
