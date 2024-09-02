package telemetry

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricType int

const (
	_ MetricType = iota
	CounterType
	GaugeType
)

type TelemetryRecord struct {
	NodeName      string `json:"nodeName"`
	PeerID        string `json:"peerId"`
	StatusVersion string `json:"statusVersion"`
	DeviceType    string `json:"deviceType"`
}

type ProcessTelemetryRequest func(ctx context.Context, data interface{})

type MetricPayload struct {
	Labels map[string]string
	Name   string
	Value  float64
}

type Metric struct {
	typ    MetricType
	labels map[string]string
}

type PrometheusMetrics struct {
	metrics         map[string]Metric
	process         ProcessTelemetryRequest
	telemetryRecord TelemetryRecord
}

func NewPrometheusMetrics(process ProcessTelemetryRequest, tc TelemetryRecord) *PrometheusMetrics {
	return &PrometheusMetrics{
		metrics:         make(map[string]Metric),
		process:         process,
		telemetryRecord: tc,
	}
}

func (pm *PrometheusMetrics) Register(name string, typ MetricType, labels prometheus.Labels) {
	pm.metrics[name] = Metric{typ, labels}
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

			pm.ToTelemetryRequest(p)
		}
	}

}

func (pm *PrometheusMetrics) ToTelemetryRequest(p MetricPayload) error {
	postBody := map[string]interface{}{
		"value":         p.Value,
		"name":          p.Name,
		"labels":        p.Labels,
		"nodeName":      pm.telemetryRecord.NodeName,
		"deviceType":    pm.telemetryRecord.DeviceType,
		"peerId":        pm.telemetryRecord.PeerID,
		"statusVersion": pm.telemetryRecord.StatusVersion,
		"timestamp":     time.Now().Unix(),
	}

	telemtryData, err := json.Marshal(postBody)
	if err != nil {
		return err
	}

	rawData := json.RawMessage(telemtryData)

	wrap := PrometheusMetricWrapper{
		Typ:  "PrometheusMetric",
		Data: &rawData,
	}

	pm.process(context.Background(), wrap)
	return nil
}
