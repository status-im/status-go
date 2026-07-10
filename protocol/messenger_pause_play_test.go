package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetPausedUpdatesMessengerFlag(t *testing.T) {
	m := &Messenger{}

	m.SetPaused(true)
	require.True(t, m.isPaused())

	m.SetPaused(false)
	require.False(t, m.isPaused())
}

func TestStoreNodeCommunityRequestsRespectPausableFlag(t *testing.T) {
	m := &Messenger{}
	m.paused.Store(true)

	manager := &StoreNodeRequestManager{messenger: m}

	pausableRequest := &storeNodeRequest{
		manager: manager,
		requestID: storeNodeRequestID{
			RequestType: storeNodeCommunityRequest,
			DataID:      "0x01",
		},
		config: StoreNodeRequestConfig{
			Pausable:          true,
			StopWhenDataFound: true,
			FurtherPageSize:   50,
		},
	}

	shouldContinue, nextPageSize := pausableRequest.shouldFetchNextPage(12)
	require.False(t, shouldContinue)
	require.Zero(t, nextPageSize)
	require.Equal(t, 0, pausableRequest.result.stats.FetchedPagesCount)
	require.Equal(t, 0, pausableRequest.result.stats.FetchedEnvelopesCount)

	nonPausableRequest := &storeNodeRequest{
		manager: manager,
		requestID: storeNodeRequestID{
			RequestType: storeNodeCommunityRequest,
			DataID:      "0x02",
		},
		config: StoreNodeRequestConfig{
			Pausable:          false,
			StopWhenDataFound: true,
			FurtherPageSize:   50,
		},
	}

	shouldContinue, nextPageSize = nonPausableRequest.shouldFetchNextPage(12)
	require.True(t, shouldContinue)
	require.Equal(t, uint64(50), nextPageSize)
	require.Equal(t, 1, nonPausableRequest.result.stats.FetchedPagesCount)
	require.Equal(t, 12, nonPausableRequest.result.stats.FetchedEnvelopesCount)
}
