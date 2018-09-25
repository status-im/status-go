package dedup

import (
	"crypto/rand"

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

func mustGenerateRandomBytes() []byte {
	c := 2048
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
