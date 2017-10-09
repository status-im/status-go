package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/status-im/status-go/geth/log"
)

// JSONFile returns a log.Metric which writes a series of batch entries into a json file.
func JSONFile(filename string, saveDir string, maxFileSizeEach int, maxBatchPerWrite int, maxwait time.Duration) *log.BatchEmitter {
	return log.BatchEmit(maxBatchPerWrite, maxwait, func(entries []log.Entry) error {
		var targetFile string
		var index int

		targetName := filename

		for {
			targetFile = path.Join(saveDir, targetName)

			stat, err := os.Stat(targetFile)
			if err != nil {
				break
			}

			if int(stat.Size()) >= maxFileSizeEach {
				index++
				targetName = fmt.Sprintf("%s_%d", filename, index)
				continue
			}

			break
		}

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
	})
}
