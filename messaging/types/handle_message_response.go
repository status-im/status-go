package types

import cryptotypes "github.com/status-im/status-go/crypto/types"

type HandleMessageResponse struct {
	StatusMessages  []*Message
	AckedMessageIDs []cryptotypes.HexBytes
}
