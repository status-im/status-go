// Package whisper collects Whisper envelope metrics using expvar.
package whisper

import (
	"github.com/ethereum/go-ethereum/metrics"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

var (
	envelopeCounter    = metrics.NewRegisteredCounter("whisper/Envelope", nil)
	envelopeNewCounter = metrics.NewRegisteredCounter("whisper/EnvelopeNew", nil)
	envelopeMeter      = metrics.NewRegisteredMeter("whisper/EnvelopeSize", nil)
)

// EnvelopeTracer traces incoming envelopes.
type EnvelopeTracer struct{}

// Trace is called for every incoming envelope.
func (t *EnvelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	envelopeCounter.Inc(1)
	if envelope.IsNew {
		envelopeNewCounter.Inc(1)
	}
	envelopeMeter.Mark(int64(envelope.Size))
}
