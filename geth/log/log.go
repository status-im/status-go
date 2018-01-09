/*Package log implements logger for status-go.

This logger handles two loggers - it's own and ethereum-go logger.
Both are used as "singletons" - using global shared variables.

Usage

First, import package into your code:

    import "github.com/status-im/status-go/geth/log

Then simply use `Info/Error/Debug/etc` functions to log at desired level:

    log.Info("Info message")
    log.Debug("Debug message")
    log.Error("Error message")

Slightly more complicated logging:

	log.Warn("abnormal conn rate", "rate", curRate, "low", lowRate, "high", highRate)

Note, in this case parameters should be in in pairs (key, value).

This logger is based upon log15-logger, so see its documentation for advanced usage: https://github.com/inconshreveable/log15


Initialization

By default logger is set to log to stdout with Error level via `init()` function.
You may change both level and file output by `log.SetLevel()` and `log.SetLogFile()` functions:

	log.SetLevel("DEBUG")
	log.SetLogFile("/path/to/geth.log")

*/
package log

//go:generate autoreadme -f

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

// Logger is a wrapper around log.Logger.
type Logger struct {
	log.Logger
	level   log.Lvl
	handler log.Handler
}

// logger is package scope instance of Logger
var logger = Logger{
	Logger:  log.New("geth", "StatusIM"),
	level:   log.LvlError,
	handler: log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
}

func init() {
	setHandler(logger.level, logger.handler)
}

// SetLevel inits status and ethereum-go logging packages,
// enabling logging and setting up proper log level.
//
// Our log levels are in form "DEBUG|ERROR|WARN|etc", while
// ethereum-go expects names in lower case: "debug|error|warn|etc".
func SetLevel(level string) {
	lvl := levelFromString(level)

	logger.level = lvl
	setHandler(lvl, logger.handler)
}

// SetLogFile configures logger to write output into file.
// This call preserves current logging level.
func SetLogFile(filename string) error {
	handler, err := log.FileHandler(filename, log.TerminalFormat(false))
	if err != nil {
		return err
	}

	logger.handler = handler
	setHandler(logger.level, handler)
	return nil
}

func levelFromString(level string) log.Lvl {
	lvl, err := log.LvlFromString(strings.ToLower(level))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Incorrect log level: %s, using defaults\n", level) // nolint: gas
		lvl = log.LvlInfo
	}
	return lvl
}

// setHandler is a helper that allows log (re)initialization
// with different level and handler. Useful for testing.
func setHandler(lvl log.Lvl, handler log.Handler) {
	h := log.LvlFilterHandler(lvl, handler)
	logger.SetHandler(h)
	log.Root().SetHandler(h) // ethereum-go logger
}

// Trace is a package scope alias for logger.Trace
func Trace(msg string, ctx ...interface{}) {
	logger.Trace(msg, ctx...)
}

// Debug is a package scope for logger.Debug
func Debug(msg string, ctx ...interface{}) {
	logger.Debug(msg, ctx...)
}

// Info is a package scope for logger.Info
func Info(msg string, ctx ...interface{}) {
	logger.Info(msg, ctx...)
}

// Warn is a package scope for logger.Warn
func Warn(msg string, ctx ...interface{}) {
	logger.Warn(msg, ctx...)
}

// Error is a package scope for logger.Error
func Error(msg string, ctx ...interface{}) {
	logger.Error(msg, ctx...)
}

// Crit is a package scope for logger.Crit
func Crit(msg string, ctx ...interface{}) {
	logger.Crit(msg, ctx...)
}
