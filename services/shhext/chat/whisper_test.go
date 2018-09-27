package chat

import (
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPublicMessageToWhisper(t *testing.T) {
	rpcMessage := &SendPublicMessageRPC{
		Chat: "test-chat",
		Sig:  "test",
	}

	payload := []byte("test")
	whisperMessage := PublicMessageToWhisper(rpcMessage, payload)

	assert.Equalf(t, uint32(10), whisperMessage.TTL, "It sets the TTL")
	assert.Equalf(t, 0.002, whisperMessage.PowTarget, "It sets the pow target")
	assert.Equalf(t, uint32(1), whisperMessage.PowTime, "It sets the pow time")
	assert.Equalf(t, whisper.TopicType{0xa4, 0xab, 0xdf, 0x64}, whisperMessage.Topic, "It sets the topic")
}

func TestDirectMessageToWhisper(t *testing.T) {
	rpcMessage := &SendDirectMessageRPC{
		PubKey: []byte("some pubkey"),
		Sig:    "test",
	}

	payload := []byte("test")
	whisperMessage := DirectMessageToWhisper(rpcMessage, payload)

	assert.Equalf(t, uint32(10), whisperMessage.TTL, "It sets the TTL")
	assert.Equalf(t, 0.002, whisperMessage.PowTarget, "It sets the pow target")
	assert.Equalf(t, uint32(1), whisperMessage.PowTime, "It sets the pow time")
	assert.Equalf(t, whisper.TopicType{0xf8, 0x94, 0x6a, 0xac}, whisperMessage.Topic, "It sets the discovery topic")
}
