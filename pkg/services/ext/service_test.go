package ext

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/pausable"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/communities"
	communitiestoken "github.com/status-im/status-go/internal/protocol/communities/token"
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

// Stands in for the messenger: the wallet's community info runs on its own goroutines,
// and a Stop (logout) can clear the service's messenger while it runs.
type fakeCommunityInfoSource struct {
	onFind  func()
	fetches int
}

func (f *fakeCommunityInfoSource) FindCommunityInfoFromDB(string) (*communities.Community, error) {
	if f.onFind != nil {
		f.onFind()
	}
	return nil, communities.ErrOrgNotFound
}

func (f *fakeCommunityInfoSource) FetchCommunity(*protocol.FetchCommunityRequest) (*communities.Community, error) {
	f.fetches++
	return nil, errors.New("messenger stopped")
}

func (f *fakeCommunityInfoSource) GetCommunityToken(string, int, string) (*communitiestoken.CommunityToken, error) {
	return nil, nil
}

// Regression test: a logout stopped the service between the community info lookup and
// the store node fetch, and the fetch then ran on a nil messenger (crash).
func TestCommunityInfoKeepsItsMessengerWhenTheServiceStops(t *testing.T) {
	svc := &Service{}
	source := &fakeCommunityInfoSource{onFind: func() { svc.setMessenger(nil) }}

	require.NotPanics(t, func() {
		_, err := fetchCommunityInfo(source, "0x0123")
		require.Error(t, err)
	})
	require.Equal(t, 1, source.fetches)

	require.NotPanics(t, func() {
		_, err := fetchCommunityInfoForCollectibles(source, "0x0123", nil)
		require.Error(t, err)
	})
	require.Equal(t, 2, source.fetches)
}

func TestCommunityInfoOfAStoppedServiceIsNotReady(t *testing.T) {
	svc := &Service{}

	_, err := svc.FetchCommunityInfo("0x0123")
	require.EqualError(t, err, "messenger not ready")

	_, err = svc.FillCollectiblesMetadata("0x0123", nil)
	require.EqualError(t, err, "messenger not ready")

	err = svc.FillCollectibleMetadata(nil, &thirdparty.FullCollectibleData{})
	require.EqualError(t, err, "messenger not ready")
}

func TestMessengerIsReadAndClearedSafelyAcrossGoroutines(t *testing.T) {
	svc := &Service{}
	messenger := &protocol.Messenger{}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = svc.currentMessenger()
			}
		}()
	}
	for j := 0; j < 1000; j++ {
		svc.setMessenger(messenger)
		svc.setMessenger(nil)
	}
	wg.Wait()
	require.Nil(t, svc.currentMessenger())
}
