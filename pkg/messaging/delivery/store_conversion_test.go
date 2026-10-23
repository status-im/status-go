package delivery

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	kcommon "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
)

func TestStoreRequestConversion(t *testing.T) {
	req := toKernelStoreRequest(&storepb.StoreQueryRequest{
		RequestId:         "r1",
		IncludeData:       true,
		PubsubTopic:       proto.String("/waku/2/rs/16/32"),
		ContentTopics:     []string{"/a/1/b/proto"},
		TimeStart:         proto.Int64(1),
		TimeEnd:           proto.Int64(2),
		MessageHashes:     [][]byte{{0xab}},
		PaginationCursor:  []byte{0xcd},
		PaginationForward: true,
		PaginationLimit:   proto.Uint64(50),
	})

	require.Equal(t, "r1", req.RequestId)
	require.Equal(t, "/waku/2/rs/16/32", req.PubsubTopic)
	require.Equal(t, []string{"/a/1/b/proto"}, *req.ContentTopics)
	require.Equal(t, []kcommon.MessageHash{"0xab"}, *req.MessageHashes)
	require.Equal(t, kcommon.MessageHash("0xcd"), *req.PaginationCursor)
	require.Equal(t, uint64(50), *req.PaginationLimit)
	require.True(t, req.IncludeData)
	require.True(t, req.PaginationForward)
}

func TestStoreResponseConversion(t *testing.T) {
	var resp kcommon.StoreQueryResponse
	require.NoError(t, json.Unmarshal([]byte(`{
		"messages": [{"messageHash": "0x01", "pubsubTopic": "/waku/2/rs/16/32",
			"message": {"payload": "aGk=", "contentTopic": "/a/1/b/proto", "timestamp": 7}}],
		"paginationCursor": "0x02"
	}`), &resp))

	messages, cursor, err := fromKernelStoreResponse(&resp)
	require.NoError(t, err)
	require.Equal(t, []byte{0x02}, cursor)
	require.Len(t, messages, 1)
	require.Equal(t, []byte{0x01}, messages[0].MessageHash)
	require.Equal(t, "/waku/2/rs/16/32", messages[0].GetPubsubTopic())
	require.Equal(t, []byte("hi"), messages[0].Message.Payload)
	require.Equal(t, int64(7), messages[0].Message.GetTimestamp())
}
