package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChannelID(t *testing.T) {
	communityID := "community-id"
	require.Equal(t, communityID, BuildChannelID(communityID))
}
