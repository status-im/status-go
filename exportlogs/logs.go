package exportlogs

import (
	"fmt"
	"io/ioutil"

	"github.com/status-im/go-ethereum/common/hexutil"
)

// Log contains actual log content and filename. If content is gzipped Compressed will be set to true.
type Log struct {
	Filename   string
	Content    []byte
	Compressed bool
}

// ExportResponse contains all available logs
type ExportResponse struct {
	Error string
	Logs  []Log
}

// ExportFromBaseFile reads log file and returns its content with some meta.
// In future can be extended to dump rotated logs.
func ExportFromBaseFile(logFile string) ExportResponse {
	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		return ExportResponse{Error: fmt.Errorf("error reading file %s: %v", logFile, err).Error()}
	}
	return ExportResponse{Logs: []Log{
		{Filename: logFile, Compressed: false, Content: hexutil.Bytes(data)},
	}}

}
