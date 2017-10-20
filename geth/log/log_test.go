package log

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const (
	trace = "trace log message\n"
	debug = "debug log message\n"
	info  = "info log message\n"
	warn  = "warning log message\n"
	err   = "error log message\n"
)

func TestLogLevels(t *testing.T) {
	var tests = []struct {
		lvl log.Lvl
		out string
	}{
		{log.LvlTrace, trace + debug + info + warn + err},
		{log.LvlDebug, debug + info + warn + err},
		{log.LvlInfo, info + warn + err},
		{log.LvlWarn, warn + err},
		{log.LvlError, err},
	}

	var buf bytes.Buffer
	// log-compatible handler that writes log in the buffer
	handler := log.FuncHandler(func(r *log.Record) error {
		_, err := buf.Write([]byte(r.Msg))
		return err
	})
	for _, test := range tests {
		buf.Reset()

		setHandler(test.lvl, handler)

		Trace(trace)
		Debug(debug)
		Info(info)
		Warn(warn)
		Error(err)

		require.Equal(t, test.out, buf.String())
	}
}

func TestLogFile(t *testing.T) {
	file, err := ioutil.TempFile("", "statusim_log_test")
	require.NoError(t, err)

	defer file.Close() //nolint: errcheck

	// setup log
	SetLevel("INFO")
	err = SetLogFile(file.Name())
	require.NoError(t, err)

	// test log output to file
	Info(info)
	Debug(debug)

	data, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	got := string(data)
	require.Contains(t, got, info)
	require.NotContains(t, got, debug)
}
