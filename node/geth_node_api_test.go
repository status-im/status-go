package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
)

func TestWakuLightModeEnabledSetsEmptyBloomFilter(t *testing.T) {
	config := params.NodeConfig{
		EnableNTPSync: true,
		WakuConfig: params.WakuConfig{
			Enabled:     true,
			LightClient: true,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config, &accounts.Manager{}))
	defer func() {
		require.NoError(t, node.Stop())
	}()

	waku := node.WakuService()
	require.NotNil(t, waku)

	bloomFilter := waku.BloomFilter()
	expectedEmptyBloomFilter := make([]byte, 64)
	require.NotNil(t, bloomFilter)
	require.Equal(t, expectedEmptyBloomFilter, bloomFilter)
}

func TestWakuLightModeEnabledSetsNilBloomFilter(t *testing.T) {
	config := params.NodeConfig{
		EnableNTPSync: true,
		WakuConfig: params.WakuConfig{
			Enabled:     true,
			LightClient: false,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config, &accounts.Manager{}))
	defer func() {
		require.NoError(t, node.Stop())
	}()

	waku := node.WakuService()
	require.NotNil(t, waku)
	require.Nil(t, waku.BloomFilter())
}
