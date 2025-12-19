package types

import (
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

type HandleMessageResponse struct {
	Messages        []*Message
	AckedMessageIDs []cryptotypes.HexBytes
}
