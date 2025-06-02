package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func TestNewEnvelope(t *testing.T) {
	version := uint32(1)
	timestamp := int64(1234567890)
	ephemeral := true
	hashBytes := []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	hashStr := hexutil.Encode(hashBytes)

	jsonStr := `{
		"wakuMessage": {
			"payload": "aGVsbG8=",
			"contentTopic": "test-topic",
			"version": 1,
			"timestamp": 1234567890,
			"meta": "bWV0YQ==",
			"ephemeral": true,
			"proof": "cHJvb2Y="
		},
		"pubsubTopic": "test-pubsub",
		"messageHash": "` + hashStr + `"
	}`

	// Create a WakuEnvelope instance and unmarshal into it
	var env WakuEnvelope
	err := json.Unmarshal([]byte(jsonStr), &env)
	assert.NoError(t, err)

	msg := env.Message()
	assert.NotNil(t, msg)
	assert.Equal(t, "test-topic", msg.ContentTopic)
	assert.Equal(t, &version, msg.Version)
	assert.Equal(t, &timestamp, msg.Timestamp)
	assert.Equal(t, ephemeral, *msg.Ephemeral)
	assert.Equal(t, "test-pubsub", env.PubsubTopic())
	assert.Equal(t, pb.ToMessageHash(hashBytes), env.Hash())

	// Test NewWakuEnvelope constructor
	newEnv := NewWakuEnvelope(msg, "new-topic", pb.ToMessageHash([]byte{0xaa, 0xbb}))
	assert.Equal(t, msg, newEnv.Message())
	assert.Equal(t, "new-topic", newEnv.PubsubTopic())
	assert.Equal(t, pb.ToMessageHash([]byte{0xaa, 0xbb}), newEnv.Hash())
}
