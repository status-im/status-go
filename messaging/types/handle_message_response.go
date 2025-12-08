package types

import (
	cryptotypes "github.com/status-im/status-go/crypto/types"
)

type HandleMessageResponse struct {
	Messages        []*Message
	AckedMessageIDs []cryptotypes.HexBytes
}
