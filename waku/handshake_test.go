package waku

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestEncodeDecodeRLP(t *testing.T) {
	pow := math.Float64bits(6.02)
	lightNodeEnabled := true
	confirmationsEnabled := true

	opts := statusOptions{
		PoWRequirement:       &pow,
		BloomFilter:          TopicToBloom(TopicType{0xaa, 0xbb, 0xcc, 0xdd}),
		LightNodeEnabled:     &lightNodeEnabled,
		ConfirmationsEnabled: &confirmationsEnabled,
		RateLimits: &RateLimits{
			IPLimits:     10,
			PeerIDLimits: 5,
			TopicLimits:  1,
		},
		TopicInterest: []TopicType{{0x01}, {0x02}, {0x03}, {0x04}},
	}
	data, err := rlp.EncodeToBytes(opts)
	require.NoError(t, err)

	var optsDecoded statusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	require.EqualValues(t, opts, optsDecoded)
}

func TestBackwardCompatibility(t *testing.T) {
	alist := []interface{}{
		[]interface{}{"0", math.Float64bits(2.05)},
	}
	data, err := rlp.EncodeToBytes(alist)
	require.NoError(t, err)

	var optsDecoded statusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	pow := math.Float64bits(2.05)
	require.EqualValues(t, statusOptions{PoWRequirement: &pow}, optsDecoded)
}

func TestForwardCompatibility(t *testing.T) {
	pow := math.Float64bits(2.05)
	alist := []interface{}{
		[]interface{}{"0", pow},
		[]interface{}{"99", uint(10)}, // some future option
	}
	data, err := rlp.EncodeToBytes(alist)
	require.NoError(t, err)

	var optsDecoded statusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	require.EqualValues(t, statusOptions{PoWRequirement: &pow}, optsDecoded)
}
