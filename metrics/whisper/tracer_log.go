// +build !metrics

// Package whisper collects Whisper envelope metrics.
package whisper

import (
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/log"
)

// EnvelopeTracer traces incoming envelopes.
type EnvelopeTracer struct{}

// Trace is called for every incoming envelope.
func (t *EnvelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	log.Debug("Received Whisper envelope", "hash", envelope.Hash, "data", envelope)
}
