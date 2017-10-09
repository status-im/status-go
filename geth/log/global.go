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
func Init(m Metrics) {
	rootlogger.ml.Lock()
	defer rootlogger.ml.Unlock()
	rootlogger.log = m
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
