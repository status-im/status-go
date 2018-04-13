package node

import (
	"testing"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"

	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

func TestWhisperLightModeEnabledSetsEmptyBloomFilter(t *testing.T) {
	config := params.NodeConfig{
		NetworkID: uint64(GetNetworkID()),
		WhisperConfig: &params.WhisperConfig{
			Enabled:     true,
			LightClient: true,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config))
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
		NetworkID: uint64(GetNetworkID()),
		WhisperConfig: &params.WhisperConfig{
			Enabled:     true,
			LightClient: false,
		},
	}
	node := New()
	require.NoError(t, node.Start(&config))
	defer func() {
		require.NoError(t, node.Stop())
	}()

	var whisper *whisper.Whisper
	require.NoError(t, node.gethService(&whisper))
	require.Nil(t, whisper.BloomFilter())
}
