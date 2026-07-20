package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChannelID(t *testing.T) {
	chatID := "community-chat-id"

	channelID := BuildChannelID(chatID)

	require.Equal(t, chatID, channelID)
}
