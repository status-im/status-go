package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/whisper/v6"
)

func TestWhisperLightModeEnabledSetsEmptyBloomFilter(t *testing.T) {
	config := params.NodeConfig{
		EnableNTPSync: true,
		WhisperConfig: params.WhisperConfig{
			Enabled:     true,
			LightClient: true,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config, &accounts.Manager{}))
	defer func() {
		require.NoError(t, node.Stop())
	}()

	var whisper *whisper.Whisper
	require.NoError(t, node.gethService(&whisper))

	bloomFilter := whisper.BloomFilter()
	expectedEmptyBloomFilter := make([]byte, 64)
	require.NotNil(t, bloomFilter)
	require.Equal(t, expectedEmptyBloomFilter, bloomFilter)
}

func TestWhisperLightModeEnabledSetsNilBloomFilter(t *testing.T) {
	config := params.NodeConfig{
		EnableNTPSync: true,
		WhisperConfig: params.WhisperConfig{
			Enabled:     true,
			LightClient: false,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config, &accounts.Manager{}))
	defer func() {
		require.NoError(t, node.Stop())
	}()

	var whisper *whisper.Whisper
	require.NoError(t, node.gethService(&whisper))
	require.Nil(t, whisper.BloomFilter())
}
