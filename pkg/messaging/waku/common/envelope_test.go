package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func TestNewWakuEnvelope(t *testing.T) {
	version := uint32(1)
	timestamp := int64(1234567890)
	msg := &pb.WakuMessage{
		Payload:      []byte("hello"),
		ContentTopic: "test-topic",
		Version:      &version,
		Timestamp:    &timestamp,
	}
	hash := pb.ToMessageHash([]byte{0xaa, 0xbb})

	env := NewWakuEnvelope(msg, "test-pubsub", hash)
	assert.Equal(t, msg, env.Message())
	assert.Equal(t, "test-pubsub", env.PubsubTopic())
	assert.Equal(t, hash, env.Hash())
}
