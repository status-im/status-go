package logutils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestPrintOrigins(t *testing.T) {
	buffer := bytes.NewBuffer(nil)

	logger := defaultLogger()
	logger.Core().(*Core).UpdateSyncer(zapcore.AddSync(buffer))

	logger.Info("hello")

	require.Contains(t, buffer.String(), "logutils/logger_test.go:17")
}
