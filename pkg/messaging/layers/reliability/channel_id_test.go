package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChannelID(t *testing.T) {
	communityID := "community-id"
	contentTopic := "community-chat"

	channelID := BuildChannelID(communityID, contentTopic)

	require.Equal(t, communityID+":"+contentTopic, channelID)
}

func TestBuildChannelIDWithoutContentTopic(t *testing.T) {
	communityID := "community-id"

	require.Equal(t, communityID, BuildChannelID(communityID, ""))
}
