package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

func TestWakuLightModeEnabledSetsEmptyBloomFilter(t *testing.T) {
	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		EnableNTPSync: true,
		WakuConfig: params.WakuConfig{
			Enabled:     true,
			LightClient: true,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	waku := statusNode.WakuService()
	require.NotNil(t, waku)

	bloomFilter := waku.BloomFilter()
	expectedEmptyBloomFilter := make([]byte, 64)
	require.NotNil(t, bloomFilter)
	require.Equal(t, expectedEmptyBloomFilter, bloomFilter)
}

func TestWakuLightModeEnabledSetsNilBloomFilter(t *testing.T) {
	statusNode, err := createAndStartStatusNode(&params.NodeConfig{
		EnableNTPSync: true,
		WakuConfig: params.WakuConfig{
			Enabled:     true,
			LightClient: false,
		},
	})
	require.NoError(t, err)
	defer func() {
		err := statusNode.Stop()
		require.NoError(t, err)
	}()

	waku := statusNode.WakuService()
	require.NotNil(t, waku)
	require.Nil(t, waku.BloomFilter())
}
