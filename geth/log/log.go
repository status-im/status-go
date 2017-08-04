package log

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/params"
)

// Logger is wrapper for custom logging
type Logger struct {
	origHandler log.Handler
	handler     log.Handler
	config      *params.NodeConfig
}

var (
	onceInitNodeLogger sync.Once
	nodeLoggerInstance *Logger
	logger             log.Logger
)

func init() {
	logger = log.New("geth", "StatusIM")
}

// SetupLogger configs logger using parameters in config
func SetupLogger(config *params.NodeConfig) (*Logger, error) {
	if !config.LogEnabled {
		return nil, nil
	}

	onceInitNodeLogger.Do(func() {
		nodeLoggerInstance = &Logger{
			config:      config,
			origHandler: log.Root().GetHandler(),
		}
		nodeLoggerInstance.handler = nodeLoggerInstance.makeLogHandler(parseLogLevel(config.LogLevel))
	})

	if err := nodeLoggerInstance.Start(); err != nil {
		return nil, err
	}

	return nodeLoggerInstance, nil
}

// SetV allows to dynamically change log level of messages being written
func (l *Logger) SetV(logLevel string) {
	log.Root().SetHandler(l.makeLogHandler(parseLogLevel(logLevel)))
}

// Start installs logger handler
func (l *Logger) Start() error {
	log.Root().SetHandler(l.handler)
	return nil
}

// Stop replaces our handler back to the original log handler
func (l *Logger) Stop() error {
	log.Root().SetHandler(l.origHandler)
	return nil
}

// Show only Status logs
func (l *Logger) DisableGeth() error {
	logger.SetHandler(l.handler)
	return nil
}

// Trace is a convenient alias for Root().Trace
func Trace(msg string, ctx ...interface{}) {
	logger.Trace(msg, ctx...)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, ctx ...interface{}) {
	logger.Debug(msg, ctx...)
}

// Info is a convenient alias for Root().Info
func Info(msg string, ctx ...interface{}) {
	logger.Info(msg, ctx...)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, ctx ...interface{}) {
	logger.Warn(msg, ctx...)
}

// Error is a convenient alias for Root().Error
func Error(msg string, ctx ...interface{}) {
	logger.Error(msg, ctx...)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, ctx ...interface{}) {
	logger.Crit(msg, ctx...)
}

// makeLogHandler creates a log handler for a given level and node configuration
func (l *Logger) makeLogHandler(lvl log.Lvl) log.Handler {
	var handler log.Handler
	logFilePath := filepath.Join(l.config.DataDir, l.config.LogFile)
	fileHandler := log.Must.FileHandler(logFilePath, log.LogfmtFormat())
	stderrHandler := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	if l.config.LogToStderr {
		handler = log.MultiHandler(
			log.LvlFilterHandler(lvl, log.CallerFileHandler(log.CallerFuncHandler(stderrHandler))),
			log.LvlFilterHandler(lvl, fileHandler))
	} else {
		handler = log.LvlFilterHandler(lvl, fileHandler)
	}

	return handler
}

// parseLogLevel parses string and returns logger.* constant
func parseLogLevel(logLevel string) log.Lvl {
	switch logLevel {
	case "ERROR":
		return log.LvlError
	case "WARNING":
		return log.LvlWarn
	case "INFO":
		return log.LvlInfo
	case "DEBUG":
		return log.LvlDebug
	case "TRACE":
		return log.LvlTrace
	}

	return log.LvlInfo
}
