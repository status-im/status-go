package waku

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
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
		keyType:       sOKTU,
	}
	data, err := rlp.EncodeToBytes(opts)
	require.NoError(t, err)

	var optsDecoded statusOptions
	err = rlp.DecodeBytes(data, &optsDecoded)
	require.NoError(t, err)
	require.EqualValues(t, opts, optsDecoded)
}

// TODO remove once key type issue is resolved.
func TestKeyTypes(t *testing.T) {
	uKeys := []uint{
		0, 1, 2, 49, 50, 256, 257, 1000, 6000,
	}

	for i, uKey := range uKeys {
		fmt.Printf("test %d, for key '%d'", i+1, uKey)

		encodeable := []interface{}{
			[]interface{}{uKey, true},
		}
		data, err := rlp.EncodeToBytes(encodeable)
		spew.Dump(data, err)

		var optsDecoded statusOptions
		err = rlp.DecodeBytes(data, &optsDecoded)
		spew.Dump(optsDecoded, err)

		println("\n----------------\n")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	pow := math.Float64bits(2.05)
	lne := true

	cs := []struct {
		Input []interface{}
		Expected statusOptions
	}{
		{
			[]interface{}{
				[]interface{}{"0", pow},
			},
			statusOptions{PoWRequirement: &pow, keyType: sOKTS},
		},
		{
			[]interface{}{
				[]interface{}{"2", true},
			},
			statusOptions{LightNodeEnabled: &lne, keyType: sOKTS},
		},
		{
			[]interface{}{
				[]interface{}{uint(2), true},
			},
			statusOptions{LightNodeEnabled: &lne, keyType: sOKTU},
		},
		{
			[]interface{}{
				[]interface{}{uint(0), pow},
			},
			statusOptions{PoWRequirement: &pow, keyType: sOKTU},
		},
		{
			[]interface{}{
				[]interface{}{"1000", true},
			},
			statusOptions{keyType: sOKTS},
		},
		{
			[]interface{}{
				[]interface{}{uint(1000), true},
			},
			statusOptions{keyType: sOKTU},
		},
	}

	for i, c := range cs {
		failMsg := fmt.Sprintf("test '%d'", i+1)

		data, err := rlp.EncodeToBytes(c.Input)
		require.NoError(t, err, failMsg)

		var optsDecoded statusOptions
		err = rlp.DecodeBytes(data, &optsDecoded)
		require.NoError(t, err, failMsg)

		require.EqualValues(t, c.Expected, optsDecoded, failMsg)
	}
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

func TestInitRLPKeyFields(t *testing.T) {
	ifk := map[int]statusOptionKey{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		5: 5,
	}
	kfi := map[statusOptionKey]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		5: 5,
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
