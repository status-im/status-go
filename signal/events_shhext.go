package signal

import "github.com/ethereum/go-ethereum/common"

const (
	// EventEnvelopeSent is triggered when envelope was sent at least to a one peer.
	EventEnvelopeSent = "envelope.sent"

	// EventEnvelopeExpired is triggered when envelop was dropped by a whisper without being sent
	// to any peer
	EventEnvelopeExpired = "envelope.expired"
)

// EnvelopeSignal includes hash of the envelope.
type EnvelopeSignal struct {
	Hash common.Hash `json:"hash"`
}

// SendEnvelopeSent triggered when envelope delivered at least to 1 peer.
func SendEnvelopeSent(hash common.Hash) {
	sendSignal(Envelope{
		Type:  EventEnvelopeSent,
		Event: EnvelopeSignal{hash},
	})
}

// SendEnvelopeExpired triggered when envelope delivered at least to 1 peer.
func SendEnvelopeExpired(hash common.Hash) {
	sendSignal(Envelope{
		Type:  EventEnvelopeExpired,
		Event: EnvelopeSignal{hash},
	})
}
