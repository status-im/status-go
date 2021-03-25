package wakuv2

import (
	"encoding/hex"

	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/types"
)

func createMessagesRequest(from, to uint32, cursor []byte, topics []types.TopicType) types.MessagesRequest {
	aUUID := uuid.New()
	// uuid is 16 bytes, converted to hex it's 32 bytes as expected by types.MessagesRequest
	id := []byte(hex.EncodeToString(aUUID[:]))
	return types.MessagesRequest{
		ID:     id,
		From:   from,
		To:     to,
		Limit:  100,
		Cursor: cursor,
	}
}
