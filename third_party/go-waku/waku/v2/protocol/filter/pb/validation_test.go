package pb

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func TestValidateRequest(t *testing.T) {
	request := &FilterSubscribeRequest{}
	require.ErrorIs(t, request.Validate(), errMissingRequestID)
	request.RequestId = "test"
	request.FilterSubscribeType = FilterSubscribeRequest_SUBSCRIBE
	require.ErrorIs(t, request.Validate(), errMissingPubsubTopic)
	pubsubTopic := "test"
	request.PubsubTopic = &pubsubTopic
	require.ErrorIs(t, request.Validate(), errNoContentTopics)
	request.ContentTopics = []string{""}
	require.ErrorIs(t, request.Validate(), errEmptyContentTopics)
	request.ContentTopics[0] = "test"
	require.NoError(t, request.Validate())
}

func TestValidateResponse(t *testing.T) {
	response := FilterSubscribeResponse{}
	require.ErrorIs(t, response.Validate(), errMissingRequestID)
	response.RequestId = "test"
	require.NoError(t, response.Validate())

}

func TestValidateMessagePush(t *testing.T) {
	msgPush := &MessagePush{}
	require.ErrorIs(t, msgPush.Validate(), errMissingMessage)
	msgPush.WakuMessage = &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: "test",
	}
	require.NoError(t, msgPush.Validate())
}
