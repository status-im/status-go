package log

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/log"
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
	// log-comaptible handler that writes log in the buffer
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

		if buf.String() != test.out {
			t.Errorf("Expecting log output to be '%s', got '%s'", test.out, buf.String())
		}
	}
}
