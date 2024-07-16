package logutils

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestPrintOrigins(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	handler := log.NewTerminalHandler(os.Stderr, false)
	require.NoError(t, enableRootLog("debug", handler))
	log.Debug("hello")
	require.Contains(t, buf.String(), "logutils/logger_test.go:16")
}
