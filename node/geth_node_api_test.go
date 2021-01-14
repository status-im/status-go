package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/whisper"
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

func TestBridgeSetup(t *testing.T) {
	testCases := []struct {
		Name         string
		Skip         string
		Cfg          params.NodeConfig
		ErrorMessage string
	}{
		{
			Name: "no whisper and waku",
			Cfg: params.NodeConfig{
				BridgeConfig: params.BridgeConfig{Enabled: true},
			},
			ErrorMessage: "setup bridge: failed to get Whisper: unknown service",
		},
		{
			Name: "only whisper",
			Cfg: params.NodeConfig{
				WhisperConfig: params.WhisperConfig{
					Enabled:     true,
					LightClient: false,
				},
				BridgeConfig: params.BridgeConfig{Enabled: true},
			},
			ErrorMessage: "setup bridge: failed to get Waku: unknown service",
		},
		{
			Name: "only waku",
			Cfg: params.NodeConfig{
				WakuConfig: params.WakuConfig{
					Enabled:     true,
					LightClient: false,
				},
				BridgeConfig: params.BridgeConfig{Enabled: true},
			},
			ErrorMessage: "setup bridge: failed to get Whisper: unknown service",
		},
		{
			Name: "both",
			Skip: "This test is flaky, setting it as skip for now",
			Cfg: params.NodeConfig{
				WhisperConfig: params.WhisperConfig{
					Enabled:     true,
					LightClient: false,
				},
				WakuConfig: params.WakuConfig{
					Enabled:     true,
					LightClient: false,
				},
				BridgeConfig: params.BridgeConfig{Enabled: true},
			},
		},
	}

	for _, tc := range testCases {
		if tc.Skip != "" {
			t.Skip(tc.Skip)
			continue
		}
		t.Run(tc.Name, func(t *testing.T) {
			node := New()
			err := node.Start(&tc.Cfg, &accounts.Manager{})
			if err != nil {
				require.EqualError(t, err, tc.ErrorMessage)
			} else if tc.ErrorMessage != "" {
				t.Fatalf("expected an error: %s", tc.ErrorMessage)
			}
			require.NoError(t, node.Stop())
		})
	}
}
