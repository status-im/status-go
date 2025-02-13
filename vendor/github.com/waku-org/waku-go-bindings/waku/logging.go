package waku

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	once     sync.Once
	instance *logrus.Logger
)

// _getLogger ensures we always return the same logger instance (private function)
func _getLogger() *logrus.Logger {
	once.Do(func() {
		instance = logrus.New()
		instance.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		instance.SetLevel(logrus.DebugLevel) // Set default log level
	})
	return instance
}

// Debug logs a debug message
func Debug(msg string, args ...interface{}) {
	_getLogger().WithFields(logrus.Fields{}).Debugf(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...interface{}) {
	_getLogger().WithFields(logrus.Fields{}).Infof(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...interface{}) {
	_getLogger().WithFields(logrus.Fields{}).Errorf(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	_getLogger().WithFields(logrus.Fields{}).Warnf(msg, args...)
}
