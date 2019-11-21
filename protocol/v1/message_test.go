package protocol

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	protocol "github.com/status-im/status-go/protocol/types"
	"github.com/stretchr/testify/require"
)

var (
	testMessageBytes  = []byte(`["~#c4",["abc123","text/plain","~:public-group-user-message",154593077368201,1545930773682,["^ ","~:chat-id","testing-adamb","~:name", "test-name","~:response-to", "id","~:text","abc123"]]]`)
	testMessageStruct = Message{
		Text:      "abc123",
		ContentT:  "text/plain",
		MessageT:  "public-group-user-message",
		Clock:     154593077368201,
		Timestamp: 1545930773682,
		Content: Content{
			ChatID:     "testing-adamb",
			Text:       "abc123",
			ResponseTo: "id",
			Name:       "test-name",
		},
	}
)

func TestDecodeTransitMessage(t *testing.T) {
	val, err := decodeTransitMessage(testMessageBytes)
	require.NoError(t, err)
	require.EqualValues(t, testMessageStruct, val)
}

func BenchmarkDecodeTransitMessage(b *testing.B) {
	_, err := decodeTransitMessage(testMessageBytes)
	if err != nil {
		b.Fatalf("failed to decode message: %v", err)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, _ = decodeTransitMessage(testMessageBytes)
	}

	data, err := EncodeMessage(testMessageStruct)
	require.NoError(b, err)
	// Decode it back to a struct because, for example, map encoding is non-deterministic
	// and it is not possible to compare bytes.
	val, err := decodeTransitMessage(data)
	require.NoError(b, err)
	require.EqualValues(b, testMessageStruct, val)
}

func TestMessageID(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyBytes := crypto.FromECDSAPub(&key.PublicKey)

	data := []byte("test")
	expectedID := protocol.HexBytes(crypto.Keccak256(append(keyBytes, data...)))
	require.Equal(t, expectedID, MessageID(&key.PublicKey, data))
}

func TestTimestampInMs(t *testing.T) {
	ts := TimestampInMs(1555274502548) // random timestamp in milliseconds
	tt := ts.Time()
	require.Equal(t, tt.UnixNano(), 1555274502548*int64(time.Millisecond))
	require.Equal(t, ts, TimestampInMsFromTime(tt))
}
