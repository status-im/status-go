package log

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

// logger is package scope instance of log.Logger
var logger = log.New("geth", "StatusIM")

func init() {
	SetLevel("INFO")
}

// Init inits status and ethereum-go logging packages,
// enabling logging and setting up proper log level.
//
// Our log levels are in form "DEBUG|ERROR|WARN|etc", while
// ethereum-go expects names in lower case: "debug|error|warn|etc".
func SetLevel(level string) {
	lvl, err := log.LvlFromString(strings.ToLower(level))
	if err != nil {
		fmt.Printf("Incorrect log level: %s, using defaults\n", level)
		lvl = log.LvlInfo
	}

	initWithHandler(lvl, log.StdoutHandler)
}

// initWithHandler is a init helper that allows (re)initialization
// with different handler. Useful for testing.
func initWithHandler(lvl log.Lvl, handler log.Handler) {
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
