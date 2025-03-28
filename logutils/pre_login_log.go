package logutils

import (
	"path/filepath"
)

const (
	defaultPreLoginLogFile    = "pre_login.log"
	defaultPreLoginLogLevel   = "ERROR"
	defaultPreLoginLogEnabled = true
)

type PreLoginLogConfig struct {
	enabled bool
	level   string
	// absolute path to the log directory, it should be the same as node config's logDir
	logDir string
}

func NewPreLoginLogConfig() *PreLoginLogConfig {
	return &PreLoginLogConfig{
		enabled: defaultPreLoginLogEnabled,
		level:   defaultPreLoginLogLevel,
	}
}

func (l *PreLoginLogConfig) SetEnabled(enabled bool) {
	l.enabled = enabled
}

func (l *PreLoginLogConfig) SetLevel(level string) error {
	if _, err := LvlFromString(level); err != nil {
		return err
	}
	l.level = level
	return nil
}

func (l *PreLoginLogConfig) SetLogDir(dir string) {
	l.logDir = dir
}

func (l *PreLoginLogConfig) ConvertToLogSettings() LogSettings {
	logFile := filepath.Join(l.logDir, defaultPreLoginLogFile)
	return LogSettings{
		Enabled: l.enabled,
		Level:   l.level,
		File:    logFile,
	}
}
