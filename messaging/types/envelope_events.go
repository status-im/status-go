package types

import "github.com/status-im/status-go/v10/eth-node/types"

type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}

type EnvelopeEventsConfig struct {
	EnvelopeEventsHandler      EnvelopeEventsHandler
	MaxMessageDeliveryAttempts int
	MailServerConfirmations    bool
}
