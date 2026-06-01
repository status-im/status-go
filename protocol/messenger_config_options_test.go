package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/communities"
)

func TestWithCommunityManagerOptions(t *testing.T) {
	cfg := messengerDefaultConfig()

	opt := WithCommunityManagerOptions([]communities.ManagerOption{
		communities.WithAllowForcingCommunityMembersReevaluation(true),
	})
	require.NoError(t, opt(&cfg))
	require.Len(t, cfg.communityManagerOptions, 1)
}
