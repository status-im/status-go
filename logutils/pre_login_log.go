package logutils

import (
	"path/filepath"
)

const (
	defaultPreLoginLogFile    = "pre_login.log"
	defaultPreLoginLogLevel   = "ERROR"
	defaultPreLoginLogEnabled = true
)

type PreLoginLog struct {
	enabled bool
	level   string
	// absolute path to the log directory, it should be the same as node config's logDir
	logDir string
}

func NewPreLoginLog() *PreLoginLog {
	return &PreLoginLog{
		enabled: defaultPreLoginLogEnabled,
		level:   defaultPreLoginLogLevel,
	}
}

func (l *PreLoginLog) SetEnabled(enabled bool) {
	l.enabled = enabled
}

func (l *PreLoginLog) SetLevel(level string) error {
	if _, err := LvlFromString(level); err != nil {
		return err
	}
	l.level = level
	return nil
}

func (l *PreLoginLog) SetLogDir(dir string) {
	l.logDir = dir
}

func (l *PreLoginLog) Settings() LogSettings {
	logFile := filepath.Join(l.logDir, defaultPreLoginLogFile)
	return LogSettings{
		Enabled: l.enabled,
		Level:   l.level,
		File:    logFile,
	}
}
