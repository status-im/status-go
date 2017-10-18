package log

import (
	"errors"
	"sync"
)

// vars
var (
	ErrMetricNotInitialized = errors.New("Default Metric not initialize")

	rootlogger = struct {
		ml  sync.Mutex
		log Metrics
	}{}
)

// Init initializes the main Metric handler which should be used at package level for
// logging.
func Init(ms ...Metrics) {
	if len(ms) == 0 {
		return
	}

	rootlogger.ml.Lock()
	defer rootlogger.ml.Unlock()

	if len(ms) == 1 {
		rootlogger.log = ms[0]
		return
	}

	rootlogger.log = New(ms, nil)
}

// Send delivers giving Entry into underline selected Metric.
func Send(en Entry) error {
	rootlogger.ml.Lock()
	defer rootlogger.ml.Unlock()
	if rootlogger.log != nil {
		return rootlogger.log.Emit(en)
	}

	return ErrMetricNotInitialized
}
