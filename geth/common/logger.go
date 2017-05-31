package common

import (
	"errors"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/common/logdna"
	"github.com/status-im/status-go/geth/params"
)

// Logger is wrapper for custom logging
type Logger struct {
	origHandler log.Handler
	handler     log.Handler
	config      *params.LoggerConfig
}

// errors
var (
	ErrLoggerDisabled = errors.New("logger is disabled")
)

// NewLogger configs and returns a logger using parameters in config
func NewLogger(config *params.LoggerConfig) (*Logger, error) {
	if !config.Enabled {
		return nil, ErrLoggerDisabled
	}

	return &Logger{
		config:      config,
		origHandler: log.Root().GetHandler(),
		handler:     makeLogHandler(config),
	}, nil
}

// SetV dynamically changes log level
func (l *Logger) SetV(lvl log.Lvl) {
	l.config.Level = logLevelString(lvl)
	log.Root().SetHandler(makeLogHandler(l.config))
}

// Attach starts log handlers
func (l *Logger) Attach() error {
	log.Root().SetHandler(makeLogHandler(l.config))
	return nil
}

// Detach restores original handler
func (l *Logger) Detach() error {
	log.Root().SetHandler(l.origHandler)
	return nil
}

// makeLogHandler creates a log handler for a given node configuration
func makeLogHandler(config *params.LoggerConfig) log.Handler {
	var handlers []log.Handler
	lvl := parseLogLevel(config.Level)

	if config.LogToFile {
		handler := log.Must.FileHandler(config.LogFile, log.LogfmtFormat())
		handlers = append(handlers, log.LvlFilterHandler(lvl, handler))
	}

	if config.LogToStderr {
		handler := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
		handlers = append(handlers, log.LvlFilterHandler(lvl, log.CallerFileHandler(log.CallerFuncHandler(handler))))
	}

	if config.LogToRemote {
		remoteLogger, err := logdna.NewClient(&logdna.Config{
			APIKey:        config.RemoteAPIKey,
			HostName:      config.RemoteHostName,
			AppName:       params.Version,
			FlushInterval: time.Duration(config.RemoteFlushInterval) * time.Second,
			BufferSize:    config.RemoteBufferSize,
		})
		if err == nil {
			remoteLogger.Start()
			formatter := log.TerminalFormat(true)
			handler := log.FuncHandler(func(r *log.Record) error {
				msg := formatter.Format(r)
				return remoteLogger.Log(r.Time, logLevelString(r.Lvl), string(msg[31:]))
			})
			handlers = append(handlers, log.LvlFilterHandler(lvl, handler))
		}
	}

	return log.MultiHandler(handlers...)
}

// parseLogLevel parses string and returns logger.* constant
func parseLogLevel(logLevel string) log.Lvl {
	switch logLevel {
	case "ERROR":
		return log.LvlError
	case "WARN":
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

// logLevelToString returns string name of the level
func logLevelString(l log.Lvl) string {
	switch l {
	case log.LvlTrace:
		return "TRACE"
	case log.LvlDebug:
		return "DEBUG"
	case log.LvlInfo:
		return "INFO"
	case log.LvlWarn:
		return "WARN"
	case log.LvlError:
		return "ERROR"
	case log.LvlCrit:
		return "CRIT"
	}

	return "BAD LEVEL"
}
