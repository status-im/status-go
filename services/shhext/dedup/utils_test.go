package dedup

import (
	"crypto/rand"

	protocol "github.com/status-im/status-protocol-go/v1"
	whisper "github.com/status-im/whisper/whisperv6"
)

func generateMessages(count int) []*whisper.Message {
	result := []*whisper.Message{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &whisper.Message{Payload: content})
	}
	return result
}

func generateStatusMessages(count int) []*protocol.StatusMessage {
	result := []*protocol.StatusMessage{}
	for ; count > 0; count-- {
		content := mustGenerateRandomBytes()
		result = append(result, &protocol.StatusMessage{
			DecryptedPayload: content,
			TransportMessage: &whisper.Message{},
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
