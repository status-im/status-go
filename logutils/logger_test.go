package logutils

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestPrintOrigins(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	handler := log.LogfmtHandlerWithSourceAndLevel(buf, slog.LevelDebug)
	require.NoError(t, enableRootLog("debug", handler))
	log.Debug("hello")
	require.Contains(t, buf.String(), "logutils/logger_test.go:17")
}
