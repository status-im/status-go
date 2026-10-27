package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestSkipSDS(t *testing.T) {
	require.True(t, skipSDS(protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION))
	require.False(t, skipSDS(protobuf.ApplicationMetadataMessage_CHAT_MESSAGE))
	require.False(t, skipSDS(protobuf.ApplicationMetadataMessage_EMOJI_REACTION))
	require.False(t, skipSDS(protobuf.ApplicationMetadataMessage_PIN_MESSAGE))
}
