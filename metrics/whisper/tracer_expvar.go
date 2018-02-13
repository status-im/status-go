// +build metrics,!prometheus

// Package whisper collects Whisper envelope metrics using expvar.
package whisper

import (
	"expvar"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

var (
	counter      = expvar.NewInt("envelope_counter")
	counterNew   = expvar.NewInt("envelope_new_counter")
	counterTopic = expvar.NewMap("envelope_topic_counter")
	counterPeer  = expvar.NewMap("envelope_peer_counter")
	volume       = expvar.NewInt("envelope_volume")
)

// EnvelopeTracer traces incoming envelopes.
type EnvelopeTracer struct{}

// Trace is called for every incoming envelope.
func (t *EnvelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	counter.Add(1)
	if envelope.IsNew {
		counterNew.Add(1)
	}
	counterTopic.Add(envelope.Topic.String(), 1)
	counterPeer.Add(envelope.Peer, 1)
	volume.Add(int64(envelope.Size))
}
