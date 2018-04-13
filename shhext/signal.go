package shhext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/signal"
)

// EnvelopeSignal includes hash of the envelope.
type EnvelopeSignal struct {
	Hash common.Hash `json:"hash"`
}

// EnvelopeSignalHandler sends signals when envelope is sent or expired.
type EnvelopeSignalHandler struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandler) EnvelopeSent(hash common.Hash) {
	signal.Send(signal.Envelope{
		Type:  signal.EventEnvelopeSent,
		Event: EnvelopeSignal{Hash: hash},
	})
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandler) EnvelopeExpired(hash common.Hash) {
	signal.Send(signal.Envelope{
		Type:  signal.EventEnvelopeExpired,
		Event: EnvelopeSignal{Hash: hash},
	})
}
