package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/protobuf"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestSkipSDS(t *testing.T) {
	for _, messageType := range []protobuf.ApplicationMetadataMessage_Type{
		protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
		protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
		protobuf.ApplicationMetadataMessage_COMMUNITY_CANCEL_REQUEST_TO_JOIN,
		protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_LEAVE,
		protobuf.ApplicationMetadataMessage_COMMUNITY_EVENTS_MESSAGE,
		protobuf.ApplicationMetadataMessage_COMMUNITY_EDIT_SHARED_ADDRESSES,
		protobuf.ApplicationMetadataMessage_COMMUNITY_REEVALUATE_PERMISSIONS_REQUEST,
		protobuf.ApplicationMetadataMessage_COMMUNITY_UPDATE_GRANT,
		protobuf.ApplicationMetadataMessage_COMMUNITY_ENCRYPTION_KEYS_REQUEST,
		protobuf.ApplicationMetadataMessage_COMMUNITY_MESSAGE_ARCHIVE_LINK,
		protobuf.ApplicationMetadataMessage_DELETE_COMMUNITY_MEMBER_MESSAGES,
	} {
		require.True(t, skipSDS(&RawMessage{MessageType: messageType}), messageType.String())
	}

	for _, messageType := range []protobuf.ApplicationMetadataMessage_Type{
		protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		protobuf.ApplicationMetadataMessage_EDIT_MESSAGE,
		protobuf.ApplicationMetadataMessage_DELETE_MESSAGE,
		protobuf.ApplicationMetadataMessage_PIN_MESSAGE,
		protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		protobuf.ApplicationMetadataMessage_THREAD_METADATA,
		protobuf.ApplicationMetadataMessage_DELETE_THREAD,
	} {
		require.False(t, skipSDS(&RawMessage{MessageType: messageType}), messageType.String())
	}

	require.True(t, skipSDS(&RawMessage{
		MessageType:           protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		CommunityKeyExMsgType: messagingtypes.KeyExMsgRekey,
	}))
}
