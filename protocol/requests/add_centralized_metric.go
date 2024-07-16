package requests

import (
	"errors"

	"github.com/status-im/status-go/centralizedmetrics/common"
)

var (
	ErrAddCentralizedMetricInvalidMetric = errors.New("add-centralized-metric: no metric")
)

type AddCentralizedMetric struct {
	Metric *common.Metric `json:"metric"`
}

func (a *AddCentralizedMetric) Validate() error {
	if a.Metric == nil {
		return ErrAddCentralizedMetricInvalidMetric
	}
	return a.Metric.Validate()
}
