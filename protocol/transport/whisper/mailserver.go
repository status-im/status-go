package whisper

import (
	"encoding/hex"
	"math/big"

	"github.com/google/uuid"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
)

const defaultMessagesRequestLimit = 100

func createMessagesRequest(from, to uint32, cursor []byte, topics []whispertypes.TopicType) whispertypes.MessagesRequest {
	aUUID := uuid.New()
	// uuid is 16 bytes, converted to hex it's 32 bytes as expected by whispertypes.MessagesRequest
	id := []byte(hex.EncodeToString(aUUID[:]))
	return whispertypes.MessagesRequest{
		ID:     id,
		From:   from,
		To:     to,
		Limit:  defaultMessagesRequestLimit,
		Cursor: cursor,
		Bloom:  topicsToBloom(topics...),
	}
}

func topicsToBloom(topics ...whispertypes.TopicType) []byte {
	i := new(big.Int)
	for _, topic := range topics {
		bloom := whispertypes.TopicToBloom(topic)
		i.Or(i, new(big.Int).SetBytes(bloom[:]))
	}

	combined := make([]byte, whispertypes.BloomFilterSize)
	data := i.Bytes()
	copy(combined[whispertypes.BloomFilterSize-len(data):], data[:])

	return combined
}
