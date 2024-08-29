package telemetry

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricType int

const (
	_ MetricType = iota
	CounterType
	GaugeType
)

type ToTelemetryRequest func(payload MetricPayload)

type MetricPayload struct {
	Labels map[string]string
	Name   string
	Value  float64
}

type Metric struct {
	typ                MetricType
	labels             map[string]string
	toTelemetryRequest ToTelemetryRequest
}

type PrometheusMetrics struct {
	metrics map[string]Metric
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		metrics: make(map[string]Metric),
	}
}

func (pm *PrometheusMetrics) Register(name string, typ MetricType, labels prometheus.Labels, toTelemetryRequest ToTelemetryRequest) {
	pm.metrics[name] = Metric{typ, labels, toTelemetryRequest}
}

func (pm *PrometheusMetrics) Snapshot() {
	gatherer := prometheus.DefaultGatherer
	metrics, err := gatherer.Gather()
	if err != nil {
		log.Fatalf("Failed to gather metrics: %v", err)
	}

	for _, mf := range metrics {
		metric, ok := pm.metrics[*mf.Name]
		if !ok {
			continue
		}
		if len(mf.GetMetric()) == 0 {
			continue
		}

		for _, m := range mf.GetMetric() {
			var p MetricPayload
			if metric.labels != nil {
				matchCnt := len(metric.labels)

				for name, value := range metric.labels {
					for _, label := range m.GetLabel() {
						if name == *label.Name && value == *label.Value {
							matchCnt--
						}
					}
				}

				if matchCnt > 0 {
					continue
				}
			}

			labelMap := make(map[string]string)

			for _, l := range m.GetLabel() {
				labelMap[*l.Name] = *l.Value
			}

			switch metric.typ {
			case CounterType:

				p = MetricPayload{
					Name:   *mf.Name,
					Value:  *m.Counter.Value,
					Labels: labelMap,
				}
			case GaugeType:
				p = MetricPayload{
					Name:   *mf.Name,
					Value:  *m.Gauge.Value,
					Labels: labelMap,
				}
			}

			metric.toTelemetryRequest(p)
		}
	}

}

func (pm *PrometheusMetrics) Get(name string) {

}
