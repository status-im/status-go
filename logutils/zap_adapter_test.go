package logutils

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/log"
)

func TestNewZapAdapter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.NewLogger(log.LogfmtHandler(buf))

	cfg := zap.NewDevelopmentConfig()
	adapter := NewZapAdapter(logger, cfg.Level)

	zapLogger := zap.New(adapter)

	buf.Reset()
	zapLogger.
		With(zap.Error(errors.New("some error"))).
		Error("some message with error level")
	require.Contains(t, buf.String(), `lvl=error msg="some message with error level" error="some error`)

	buf.Reset()
	zapLogger.
		With(zap.Int("counter", 100)).
		Info("some message with param", zap.String("another-field", "another-value"))
	require.Contains(t, buf.String(), `lvl=info msg="some message with param" counter=100 another-field=another-value`)

	buf.Reset()
	zapLogger.
		With(zap.Namespace("some-namespace")).
		With(zap.String("site", "SomeSite")).
		Info("some message with param")
	require.Contains(t, buf.String(), `lvl=info msg="some message with param" namespace=some-namespace site=SomeSite`)
}

func TestNewZapLoggerWithAdapter(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	logger := log.NewLogger(log.LogfmtHandler(buf))

	zapLogger, err := NewZapLoggerWithAdapter(logger)
	require.NoError(t, err)

	buf.Reset()
	zapLogger.
		With(zap.Error(errors.New("some error"))).
		Error("some message with error level")
	require.Contains(t, buf.String(), `lvl=error msg="some message with error level" error="some error`)
}

func TestZapLoggerTerminalFormat(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.NewLoggerWithOpts(log.LogfmtHandlerWithSource(buf), &log.LoggerOptions{AddSource: true, SkipCallers: 2})

	zapLogger, err := NewZapLoggerWithAdapter(logger)
	require.NoError(t, err)

	zapLogger.Info("some message with error level")
	require.Contains(t, buf.String(), `logutils/zap_adapter_test.go:65`)
}
