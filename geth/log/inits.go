package log

import (
	"errors"
	"sync"
)

// vars
var (
	ErrMetricNotInitialize = errors.New("Default Metric not initialize")

	rootMetric = struct {
		ml     sync.Mutex
		Metric Metric
	}{}
)

// InitMetric initializes the main Metric handler which should be used at package level for
// logging.
func InitMetric(m Metric) {
	rootMetric.ml.Lock()
	defer rootMetric.ml.Unlock()
	rootMetric.Metric = m
}

// Send delivers giving Entry into underline selected Metric.
func Send(en Entry) error {
	rootMetric.ml.Lock()
	defer rootMetric.ml.Unlock()
	if rootMetric.Metric != nil {
		return rootMetric.Metric.Emit(en)
	}

	return ErrMetricNotInitialize
}
