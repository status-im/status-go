package zaputil

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestJSONHexEncoder(t *testing.T) {
	encoder := NewJSONHexEncoder(zap.NewDevelopmentEncoderConfig())
	encoder.AddBinary("test-key", []byte{0x01, 0x02, 0x03})
	buf, err := encoder.EncodeEntry(zapcore.Entry{
		LoggerName: "",
		Time:       time.Now(),
		Level:      zapcore.DebugLevel,
		Message:    "",
	}, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), `"test-key":"0x010203"`)
}

func TestLoggerWithJSONHexEncoder(t *testing.T) {
	err := RegisterJSONHexEncoder()
	require.NoError(t, err)

	tmpFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{tmpFile.Name()}
	cfg.Encoding = "json-hex"
	l, err := cfg.Build()
	require.NoError(t, err)

	l.With(zap.Binary("some-field", []byte{0x01, 0x02, 0x03})).Warn("test message")
	err = l.Sync()
	require.NoError(t, err)

	data, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	require.Contains(t, string(data), "0x010203")
}

func TestConsoleHexEncoder(t *testing.T) {
	encoder := NewConsoleHexEncoder(zap.NewDevelopmentEncoderConfig())
	encoder.AddBinary("test-key", []byte{0x01, 0x02, 0x03})
	buf, err := encoder.EncodeEntry(zapcore.Entry{
		LoggerName: "",
		Time:       time.Now(),
		Level:      zapcore.DebugLevel,
		Message:    "",
	}, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), `{"test-key": "0x010203"}`)
}

func TestLoggerWithConsoleHexEncoder(t *testing.T) {
	err := RegisterConsoleHexEncoder()
	require.NoError(t, err)

	tmpFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{tmpFile.Name()}
	cfg.Encoding = "console-hex"
	l, err := cfg.Build()
	require.NoError(t, err)

	l.With(zap.Binary("some-field", []byte{0x01, 0x02, 0x03})).Warn("test message")
	err = l.Sync()
	require.NoError(t, err)

	data, err := ioutil.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	require.Contains(t, string(data), "0x010203")
}
