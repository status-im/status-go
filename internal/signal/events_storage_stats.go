package signal

const (
	// EventStorageStatsProgress is emitted once per collection step.
	EventStorageStatsProgress = "storage-stats.progress"

	// EventStorageStatsResult is emitted once per collection, with either the
	// profile or an error.
	EventStorageStatsResult = "storage-stats.result"
)

// StorageStatsProgressEvent carries a Total that is final from the first event.
type StorageStatsProgressEvent struct {
	RequestID string `json:"requestId"`
	Step      int    `json:"step"`
	Total     int    `json:"total"`
}

// StorageStatsResultEvent carries the finished profile in Data.
type StorageStatsResultEvent struct {
	RequestID string      `json:"requestId"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// SendStorageStatsProgress emits a progress update for a running collection.
func SendStorageStatsProgress(requestID string, step, total int) {
	send(EventStorageStatsProgress, StorageStatsProgressEvent{
		RequestID: requestID,
		Step:      step,
		Total:     total,
	})
}

// SendStorageStatsResult emits the outcome of a collection.
func SendStorageStatsResult(requestID string, data interface{}, err error) {
	event := StorageStatsResultEvent{
		RequestID: requestID,
		Data:      data,
	}
	if err != nil {
		event.Error = err.Error()
	}
	send(EventStorageStatsResult, event)
}
