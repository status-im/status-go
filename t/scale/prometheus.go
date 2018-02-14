package scale

import (
	"errors"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func pullMetrics(url string) (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint (errcheck)
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

func pullOldNewEnvelopesCount(url string) (old float64, new float64, err error) { // nolint (deadcode)
	metrics, err := pullMetrics(url)
	if err != nil {
		return old, new, err
	}
	envelope, ok := metrics["envelope_counter"]
	if !ok {
		return old, new, errors.New("envelope_counter metrics is not found")
	}
	for _, m := range envelope.Metric {
		var (
			isNew bool
		)
		for _, pair := range m.Label {
			if pair.GetName() == "is_new" {
				isNew = pair.GetValue() == "true"
			}
		}
		if isNew {
			new += m.GetCounter().GetValue()
		} else {
			old += m.GetCounter().GetValue()
		}
	}
	return old, new, err
}
