// +build metrics,prometheus

// Package whisper collects Whisper envelope metrics using Prometheus.
package whisper

import (
	"strconv"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	envelopeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envelope_counter",
			Help: "Envelopes counter",
		},
		[]string{"topic", "source", "is_new", "peer"},
	)
	envelopeVolume = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "envelope_volume",
			Help: "Volume of received envelopes",
		},
		[]string{"topic", "source", "is_new", "peer"},
	)
)

func init() {
	prometheus.MustRegister(envelopeCounter)
	prometheus.MustRegister(envelopeVolume)
}

// EnvelopeTracer traces incoming envelopes.
type EnvelopeTracer struct{}

// Trace is called for every incoming envelope.
func (t *EnvelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	labelValues := []string{
		envelope.Topic.String(),
		envelope.SourceString(),
		strconv.FormatBool(envelope.IsNew),
		envelope.Peer,
	}

	envelopeCounter.WithLabelValues(labelValues...).Inc()
	envelopeVolume.WithLabelValues(labelValues...).Add(float64(envelope.Size))
}
