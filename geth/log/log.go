package log

import (
	"github.com/ethereum/go-ethereum/log"
)

// Logger is wrapper for go-ethereum log
type Logger struct {
	output log.Logger
}

// Instance to a logger struct
var instance *Logger

// Trace is a convenient alias for Root().Trace
func Trace(msg string, ctx ...interface{}) {
	printLog(log.LvlTrace, msg, ctx...)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, ctx ...interface{}) {
	printLog(log.LvlDebug, msg, ctx...)
}

// Info is a convenient alias for Root().Info
func Info(msg string, ctx ...interface{}) {
	printLog(log.LvlInfo, msg, ctx...)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, ctx ...interface{}) {
	printLog(log.LvlWarn, msg, ctx...)
}

// Error is a convenient alias for Root().Error
func Error(msg string, ctx ...interface{}) {
	printLog(log.LvlError, msg, ctx...)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, ctx ...interface{}) {
	printLog(log.LvlCrit, msg, ctx...)
}

// outputs the log to a given log config level
func printLog(lvl log.Lvl, msg string, ctx ...interface{}) {
	if instance == nil {
		instance = &Logger{
			output: log.New("geth", "StatusIM"),
		}
		instance.output.SetHandler(log.StdoutHandler)
	}

	switch lvl {

	case log.LvlError:
		instance.output.Error(msg, ctx...)
		break
	case log.LvlWarn:
		instance.output.Warn(msg, ctx...)
		break
	case log.LvlInfo:
		instance.output.Info(msg, ctx...)
		break
	case log.LvlDebug:
		instance.output.Debug(msg, ctx...)
		break
	case log.LvlTrace:
		instance.output.Trace(msg, ctx...)
		break
	default:
		instance.output.Info(msg, ctx...)

	}
}
