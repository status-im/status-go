package dedup

import (
	"crypto/rand"

	"github.com/status-im/status-go/eth-node/types"
)

func generateMessages(count int) []*types.Message {
	result := []*types.Message{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &types.Message{Payload: content})
	}
	return result
}

func generateDedupMessages(count int) []*DeduplicateMessage {
	result := []*DeduplicateMessage{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &DeduplicateMessage{
			Metadata: Metadata{},
			Message:  &types.Message{Payload: content},
		})
	}
	return result
}

func mustGenerateRandomBytes() []byte {
	c := 2048
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
