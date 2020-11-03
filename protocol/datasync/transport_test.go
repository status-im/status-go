package datasync

import (
	"testing"

	"github.com/vacp2p/mvds/protobuf"

	"github.com/stretchr/testify/require"
)

func TestSplitPayloadInBatches(t *testing.T) {
	payload := &protobuf.Payload{Acks: [][]byte{{0x1}}}

	response := splitPayloadInBatches(payload, 100)
	require.NotNil(t, response)
	require.Len(t, response, 1)

	payload = &protobuf.Payload{Acks: [][]byte{{0x1}, {0x2}, {0x3}, {0x4}}}
	// 1 is the maximum size of the actual ack, + the tag size + 1, the length of the field
	response = splitPayloadInBatches(payload, 1+payloadTagSize+1)
	require.NotNil(t, response)
	require.Len(t, response, 4)

	payload = &protobuf.Payload{Offers: [][]byte{{0x1}, {0x2}, {0x3}, {0x4}}}
	response = splitPayloadInBatches(payload, 1+payloadTagSize+1)
	require.NotNil(t, response)
	require.Len(t, response, 4)

	payload = &protobuf.Payload{Requests: [][]byte{{0x1}, {0x2}, {0x3}, {0x4}}}
	response = splitPayloadInBatches(payload, 1+payloadTagSize+1)
	require.NotNil(t, response)
	require.Len(t, response, 4)

	payload = &protobuf.Payload{Messages: []*protobuf.Message{
		{GroupId: []byte{0x1}, Timestamp: 1, Body: []byte{0x1}},
		{GroupId: []byte{0x2}, Timestamp: 1, Body: []byte{0x2}},
		{GroupId: []byte{0x3}, Timestamp: 1, Body: []byte{0x3}},
		{GroupId: []byte{0x4}, Timestamp: 1, Body: []byte{0x4}},
	},
	}
	// 1 for the size of Messages + 2 for the sizes of the repeated MessageFields fields + 10 for the worst size of timestamps + 1 for the size of the body + 1 for the size of group id
	response = splitPayloadInBatches(payload, 1+payloadTagSize+2+timestampPayloadSize+1+1)
	require.NotNil(t, response)
	require.Len(t, response, 4)
}
