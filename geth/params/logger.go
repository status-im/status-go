package params

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Logger is wrapper for custom logging
type Logger struct {
	sync.Mutex
	logFile  *os.File
	observer chan string
	started  chan struct{}
	stopped  chan struct{}
	stopFlag bool
}

var onceStartLogger sync.Once

// SetupLogger configs logger using parameters in config
func SetupLogger(config *NodeConfig) (nodeLogger *Logger, err error) {
	if !config.LogEnabled {
		return nil, nil
	}

	nodeLogger = &Logger{
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}

	onceStartLogger.Do(func() {
		err = nodeLogger.createAndStartLogger(config)
	})

	return
}

// SetV allows to dynamically change log level of messages being written
func (l *Logger) SetV(logLevel string) {
	glog.SetV(parseLogLevel(logLevel))
}

// Stop marks logger as stopped, forcing to relinquish hold
// on os.Stderr and restore it back to the original
func (l *Logger) Stop() (stopped chan struct{}) {
	l.Lock()
	defer l.Unlock()

	l.stopFlag = true
	stopped = l.stopped
	return
}

// Observe registers extra writer where logs should be written to.
// This method is used in unit tests, and should NOT be relied upon otherwise.
func (l *Logger) Observe(observer chan string) (started chan struct{}) {
	l.observer = observer
	started = l.started
	return
}

// createAndStartLogger initializes and starts logger by replacing os.Stderr with custom writer.
// Custom writer intercepts all requests to os.Stderr, then forwards to multiple readers, which
// include log file and the original os.Stderr (so that logs output on screen as well)
func (l *Logger) createAndStartLogger(config *NodeConfig) error {
	// customize glog
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)
	glog.SetV(parseLogLevel(config.LogLevel))

	// create log file
	logFile, err := os.OpenFile(filepath.Join(config.DataDir, config.LogFile), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	// inject reader to pipe all writes to Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	// replace Stderr
	origStderr := os.Stderr
	os.Stderr = w
	scanner := bufio.NewScanner(r)

	// configure writer, send to the original os.Stderr and log file
	logWriter := io.MultiWriter(origStderr, logFile)

	go func() {
		defer func() { // restore original Stderr
			os.Stderr = origStderr
			logFile.Close()
			close(l.stopped)
		}()

		// notify observer that it can start polling (unit test, normally)
		close(l.started)

		for scanner.Scan() {
			fmt.Fprintln(logWriter, scanner.Text())

			if l.observer != nil {
				l.observer <- scanner.Text()
			}

			// allow to restore original os.Stderr if logger is stopped
			if l.stopFlag {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(origStderr, "error reading logs: %v\n", err)
		}
	}()

	return nil
}

// parseLogLevel parses string and returns logger.* constant
func parseLogLevel(logLevel string) int {
	switch logLevel {
	case "ERROR":
		return logger.Error
	case "WARNING":
		return logger.Warn
	case "INFO":
		return logger.Info
	case "DEBUG":
		return logger.Debug
	case "DETAIL":
		return logger.Detail
	}

	return logger.Info
}
