package v0

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/waku/common"
)

func TestEncodeDecodeRLP(t *testing.T) {
	initRLPKeyFields()
	pow := math.Float64bits(6.02)
	lightNodeEnabled := true
	confirmationsEnabled := true

	opts := StatusOptions{
		PoWRequirement:       &pow,
		BloomFilter:          common.TopicToBloom(common.TopicType{0xaa, 0xbb, 0xcc, 0xdd}),
		LightNodeEnabled:     &lightNodeEnabled,
		ConfirmationsEnabled: &confirmationsEnabled,
		RateLimits: &common.RateLimits{
			IPLimits:     10,
			PeerIDLimits: 5,
			TopicLimits:  1,
		},
		TopicInterest: []common.TopicType{{0x01}, {0x02}, {0x03}, {0x04}},
	}
	data, err := rlp.EncodeToBytes(opts)
	require.NoError(t, err)

	var optsDecoded StatusOptions
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

	var optsDecoded StatusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	pow := math.Float64bits(2.05)
	require.EqualValues(t, StatusOptions{PoWRequirement: &pow}, optsDecoded)
}

func TestForwardCompatibility(t *testing.T) {
	pow := math.Float64bits(2.05)
	alist := []interface{}{
		[]interface{}{"0", pow},
		[]interface{}{"99", uint(10)}, // some future option
	}
	data, err := rlp.EncodeToBytes(alist)
	require.NoError(t, err)

	var optsDecoded StatusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	require.EqualValues(t, StatusOptions{PoWRequirement: &pow}, optsDecoded)
}

func TestInitRLPKeyFields(t *testing.T) {
	ifk := map[int]statusOptionKey{
		0: "0",
		1: "1",
		2: "2",
		3: "3",
		4: "4",
		5: "5",
	}
	kfi := map[statusOptionKey]int{
		"0": 0,
		"1": 1,
		"2": 2,
		"3": 3,
		"4": 4,
		"5": 5,
	}

	// Test that the kfi length matches the inited global keyFieldIdx length
	require.Equal(t, len(kfi), len(keyFieldIdx))

	// Test that each index of the kfi values matches the inited global keyFieldIdx of the same index
	for k, v := range kfi {
		require.Exactly(t, v, keyFieldIdx[k])
	}

	// Test that each index of the inited global keyFieldIdx values matches kfi values of the same index
	for k, v := range keyFieldIdx {
		require.Exactly(t, v, kfi[k])
	}

	// Test that the ifk length matches the inited global idxFieldKey length
	require.Equal(t, len(ifk), len(idxFieldKey))

	// Test that each index of the ifk values matches the inited global idxFieldKey of the same index
	for k, v := range ifk {
		require.Exactly(t, v, idxFieldKey[k])
	}

	// Test that each index of the inited global idxFieldKey values matches ifk values of the same index
	for k, v := range idxFieldKey {
		require.Exactly(t, v, ifk[k])
	}
}
