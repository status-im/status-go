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
}
