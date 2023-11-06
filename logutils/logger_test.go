package logutils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestPrintOrigins(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	handler := log.StreamHandler(buf, log.TerminalFormat(false))
	require.NoError(t, enableRootLog("debug", handler))
	log.Debug("hello")
	require.Contains(t, buf.String(), "logutils/logger_test.go:16")
}
