package logutils

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestOverrideCoreWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		settings    LogSettings
		expectError bool
	}{
		{
			name: "disabled logging",
			settings: LogSettings{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			settings: LogSettings{
				Enabled: true,
				Level:   "invalid",
			},
			expectError: true,
		},
		{
			name: "valid log level",
			settings: LogSettings{
				Enabled: true,
				Level:   "info",
			},
			expectError: false,
		},
		{
			name: "mobile system logging",
			settings: LogSettings{
				Enabled:      true,
				MobileSystem: true,
			},
			expectError: false,
		},
		{
			name: "file logging with rotation",
			settings: LogSettings{
				Enabled:         true,
				File:            "test.log",
				MaxSize:         10,
				MaxBackups:      0,
				CompressRotated: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := NewCore(
				defaultEncoder(),
				zapcore.AddSync(io.Discard),
				zap.NewAtomicLevelAt(zap.InfoLevel),
			)
			err := overrideCoreWithConfig(core, tt.settings)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
