package shhext

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/signal"
)

// EnvelopeSentSignal includes hash of the sent envelope.
type EnvelopeSentSignal struct {
	Hash common.Hash `json:"hash"`
}

// SendEnvelopeSentSignal sends an envelope.sent signal with hash of the envelope.
func SendEnvelopeSentSignal(hash common.Hash) {
	signal.Send(signal.Envelope{
		Type: signal.EventEnvelopeSent,
		Event: EnvelopeSentSignal{
			Hash: hash,
		},
	})
}
