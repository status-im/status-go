package ext

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/pausable"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func TestServicePauseResumeBackgroundWithNilMessenger(t *testing.T) {
	svc := &Service{}
	require.NoError(t, svc.Pause())
	require.Equal(t, pausable.ServiceStatePaused, svc.PausableState())
	require.NoError(t, svc.Resume())
	require.Equal(t, pausable.ServiceStateRunning, svc.PausableState())
}

func TestServicePauseResumeBackgroundWithMessenger(t *testing.T) {
	svc := &Service{
		messenger: &protocol.Messenger{},
	}
	require.NoError(t, svc.Pause())
	require.Equal(t, pausable.ServiceStatePaused, svc.PausableState())
	require.NoError(t, svc.Resume())
	require.Equal(t, pausable.ServiceStateRunning, svc.PausableState())
}

// Regression test: right after a profile sync the community description
// (token metadata) can arrive before the community_tokens rows are synced,
// so the community token lookup legitimately returns nil. Filling metadata
// must tolerate that instead of crashing the node.
func TestFillCollectibleMetadataWithMissingCommunityToken(t *testing.T) {
	collectible := &thirdparty.FullCollectibleData{
		CollectibleData: thirdparty.CollectibleData{
			CommunityID: "0x0123",
		},
	}
	tokenMetadata := &protobuf.CommunityTokenMetadata{
		Name:        "Community Collectible",
		Description: "desc",
	}

	require.NotPanics(t, func() {
		fillCollectibleMetadata(collectible, nil, tokenMetadata, nil, nil)
	})

	require.Equal(t, "Community Collectible", collectible.CollectibleData.Name)
	require.False(t, collectible.CollectibleData.Soulbound)
	require.NotNil(t, collectible.CollectionData)
}
