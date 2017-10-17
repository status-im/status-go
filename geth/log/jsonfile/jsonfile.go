package jsonfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/status-im/status-go/geth/log"
)

// JSON returns a log.Metric which writes a series of batch entries into a json file.
func JSON(targetFile string, maxBatchPerWrite int, maxwait time.Duration) (*log.BatchEmitter, error) {
	// If the directory does not exists, create it first.
	dir := filepath.Dir(targetFile)
	if dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}

	return log.BatchEmit(maxBatchPerWrite, maxwait, func(entries []log.Entry) error {
		logFile, err := os.OpenFile(targetFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		defer logFile.Close()

		encoder := json.NewEncoder(logFile)

		for _, item := range entries {
			if err := encoder.Encode(item); err != nil {
				return err
			}
		}

		logFile.Sync()

		return nil
	}), nil
}
