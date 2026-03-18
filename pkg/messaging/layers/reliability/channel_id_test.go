package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

func TestBuildChannelID(t *testing.T) {
	communityID := []byte("community-id")
	contentTopic := "content-topic"

	channelID := BuildChannelID(communityID, contentTopic)

	require.Equal(t, cryptotypes.EncodeHex(communityID)+"|"+contentTopic, channelID)
}
