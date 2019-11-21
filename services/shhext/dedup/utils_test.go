package dedup

import (
	"crypto/rand"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
)

func generateMessages(count int) []*whispertypes.Message {
	result := []*whispertypes.Message{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &whispertypes.Message{Payload: content})
	}
	return result
}

func generateDedupMessages(count int) []*DeduplicateMessage {
	result := []*DeduplicateMessage{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &DeduplicateMessage{
			Metadata: Metadata{},
			Message:  &whispertypes.Message{Payload: content},
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
