package logutils

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestGethAdapter(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	buffer := bytes.NewBuffer(nil)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer),
		level,
	)
	logger := zap.New(core)

	log.Root().SetHandler(gethAdapter(logger))

	log.Debug("should not be printed, as it's below the log level")
	require.Empty(t, buffer.String())

	buffer.Reset()
	log.Info("should be printed")
	require.Regexp(t, `INFO\s+'INFO\s*\[.*\]\s*should be printed '`, buffer.String())

	buffer.Reset()
	level.SetLevel(zap.DebugLevel)
	log.Debug("should be printed with context", "value1", 12345, "value2", "string")
	require.Regexp(t, `DEBUG\s+'DEBUG\s*\[.*\]\s*should be printed with context\s+value1=12345\s+value2=string'`, buffer.String())

	buffer.Reset()
	log.Trace("should be skipped")
	require.Empty(t, buffer.String())
}
