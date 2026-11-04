package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestEnvelope(t *testing.T) {
	e := NewEnvelope(
		&pb.WakuMessage{ContentTopic: "ContentTopic"},
		*utils.GetUnixEpoch(),
		"test",
	)

	msg := e.Message()
	require.Equal(t, "ContentTopic", msg.ContentTopic)

	topic := e.PubsubTopic()
	require.Equal(t, "test", topic)
	hash := e.Hash()

	require.Equal(
		t,
		pb.ToMessageHash([]byte{0x91, 0x0, 0xe4, 0xa5, 0xcf, 0xf7, 0x19, 0x27, 0x49, 0x81, 0x66, 0xb3, 0xdf, 0xc7, 0xa6, 0x31, 0xf0, 0x87, 0xc7, 0x29, 0xb4, 0x28, 0x83, 0xb9, 0x5c, 0x31, 0x25, 0x33, 0x3, 0xc9, 0x7, 0x95}),
		hash,
	)
}
