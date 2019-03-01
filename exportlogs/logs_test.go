package exportlogs

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportLogs(t *testing.T) {
	tempf, err := ioutil.TempFile("", "test-dump-logs")
	require.NoError(t, err)
	logs := "first line\nsecond line\n"
	n, err := fmt.Fprintf(tempf, logs)
	require.NoError(t, err)
	require.Equal(t, len(logs), n)
	response := ExportFromBaseFile(tempf.Name())
	require.Empty(t, response.Error)
	require.Len(t, response.Logs, 1)
	log := response.Logs[0]
	require.Equal(t, false, log.Compressed)
	require.Equal(t, tempf.Name(), log.Filename)
	require.Equal(t, logs, string(log.Content))
}

func TestExportLogsNoFileError(t *testing.T) {
	response := ExportFromBaseFile("doesnt-exist")
	require.Equal(t, "error reading file doesnt-exist: open doesnt-exist: no such file or directory", response.Error)
}
