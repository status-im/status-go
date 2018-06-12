package signal

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	// EventEnvelopeSent is triggered when envelope was sent at least to a one peer.
	EventEnvelopeSent = "envelope.sent"

	// EventEnvelopeExpired is triggered when envelop was dropped by a whisper without being sent
	// to any peer
	EventEnvelopeExpired = "envelope.expired"

	// EventMailServerAck is triggered when whisper receives a message ack from the mailserver
	EventMailServerAck = "mailserver.ack"
)

// EnvelopeSignal includes hash of the envelope.
type EnvelopeSignal struct {
	Hash common.Hash `json:"hash"`
}

// SendEnvelopeSent triggered when envelope delivered at least to 1 peer.
func SendEnvelopeSent(hash common.Hash) {
	send(EventEnvelopeSent, EnvelopeSignal{hash})
}

// SendEnvelopeExpired triggered when envelope delivered at least to 1 peer.
func SendEnvelopeExpired(hash common.Hash) {
	send(EventEnvelopeExpired, EnvelopeSignal{hash})
}

// SendEnvelopeExpired triggered when envelope delivered at least to 1 peer.
func SendMailServerAck(hash common.Hash) {
	send(EventMailServerAck, EnvelopeSignal{hash})
}
