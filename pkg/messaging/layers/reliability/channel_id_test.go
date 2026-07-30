package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChannelID(t *testing.T) {
	communityID := "community-id"

	channelID := BuildChannelID(communityID)

	require.Equal(t, communityID, channelID)
}
