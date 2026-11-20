package types

import (
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

type HandleMessageResponse struct {
	Messages        []*Message
	AckedMessageIDs []cryptotypes.HexBytes
	// ConsumedEnvelopeHashes identifies control-message envelopes to confirm
	// even though they produce no application messages.
	ConsumedEnvelopeHashes [][]byte
	// RejectedEnvelopeHashes are consumed for transport deduplication but must
	// not count as evidence that an SDS dependency was successfully processed.
	RejectedEnvelopeHashes [][]byte
	// RetryableMessages contains stored hash-ratchet envelopes whose SDS
	// validation failed transiently and should be retried by the caller.
	RetryableMessages []*ReceivedMessage
}
