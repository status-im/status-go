// +build !metrics

// Package whisper collects Whisper envelope metrics.
package whisper

import (
	"github.com/ethereum/go-ethereum/log"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// EnvelopeTracer traces incoming envelopes.
type EnvelopeTracer struct{}

// Trace is called for every incoming envelope.
func (t *EnvelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	logger := log.New("package", "status-go/metrics/whisper.EnvelopeTracer")
	logger.Debug("Received Whisper envelope", "hash", envelope.Hash, "data", envelope)
}
