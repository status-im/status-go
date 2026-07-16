package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreNodeRequestCommunityDescriptionFreshness(t *testing.T) {
	request := &storeNodeRequest{
		minimumDataClock: 10,
		config:           defaultStoreNodeRequestConfig(),
	}

	require.False(t, request.hasFoundCommunityDescription(10))
	require.True(t, request.hasFoundCommunityDescription(11))

	request.config = buildStoreNodeRequestConfig([]StoreNodeRequestOption{
		WithRequireNewerCommunityDescription(false),
	})
	require.True(t, request.hasFoundCommunityDescription(10))
}
