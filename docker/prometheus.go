package scale

import (
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func getMetrics(url string) (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(resp.Body)
}

func getOldNewEnvelopesCount(url string) (old float64, new float64, err error) {
	metrics, err := getMetrics(url)
	if err != nil {
		return old, new, err
	}
	envelope := metrics["envelope_counter"]
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
