package node

import (
	"testing"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/params"

	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

func TestWhisperLightModeEnabledSetsEmptyBloomFilter(t *testing.T) {
	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	config.WhisperConfig.LightClient = true

	node, nodeErr := MakeNode(config)
	require.NoError(t, nodeErr)
	require.NoError(t, node.Start())
	defer func() {
		err := node.Stop()
		require.NoError(t, err)
	}()

	var whisper *whisper.Whisper
	err = node.Service(&whisper)
	require.NoError(t, err)

	bloomFilter := whisper.BloomFilter()
	expectedEmptyBloomFilter := make([]byte, 64)
	require.NotNil(t, bloomFilter)
	require.Equal(t, expectedEmptyBloomFilter, bloomFilter)
}

func TestWhisperLightModeEnabledSetsNilBloomFilter(t *testing.T) {
	config, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	config.WhisperConfig.LightClient = false

	node, nodeErr := MakeNode(config)
	require.NoError(t, nodeErr)
	require.NoError(t, node.Start())
	defer func() {
		err := node.Stop()
		require.NoError(t, err)
	}()

	var whisper *whisper.Whisper
	err = node.Service(&whisper)
	require.NoError(t, err)
	require.Nil(t, whisper.BloomFilter())
}
