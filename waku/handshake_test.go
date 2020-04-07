package waku

import (
	"math"
	"testing"
	"reflect"
	"strconv"
	"strings"

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

func TestStatusOptionKeys(t *testing.T) {
	o := statusOptions{}

	kfi := make(map[statusOptionKey]int)
	ifk := make(map[int]statusOptionKey)

	v := reflect.ValueOf(o)

	for i := 0; i < v.NumField(); i++ {
		// skip unexported fields
		if !v.Field(i).CanInterface() {
			continue
		}
		rlpTag := v.Type().Field(i).Tag.Get("rlp")
		// skip fields without rlp field tag
		if rlpTag == "" {
			continue
		}

		keys := strings.Split(rlpTag, "=")
		require.Equal(t, 2, len(keys))

		// parse keys[1] as an int
		key, err := strconv.ParseUint(keys[1], 10, 64)
		require.NoError(t, err)

		// typecast key to be of statusOptionKey type
		kfi[statusOptionKey(key)] = i
		ifk[i] = statusOptionKey(key)
	}

	// Test that the statusOptions' derived kfi length matches the global keyFieldIdx length
	require.Equal(t, len(keyFieldIdx), len(kfi))

	// Test that each index of the statusOptions' derived kfi values matches the global keyFieldIdx of the same index
	for k, v := range kfi {
		require.Equal(t, keyFieldIdx[k], v)
	}

	// Test that each index of the global keyFieldIdx values matches statusOptions' derived kfi values of the same index
	for k, v := range keyFieldIdx {
		require.Equal(t, kfi[k], v)
	}

	// Test that the statusOptions' derived ifk length matches the global idxFieldKey length
	require.Equal(t, len(idxFieldKey), len(ifk))

	// Test that each index of the statusOptions' derived ifk values matches the global idxFieldKey of the same index
	for k, v := range ifk {
		require.Equal(t, idxFieldKey[k], v)
	}

	// Test that each index of the global idxFieldKey values matches statusOptions' derived ifk values of the same index
	for k, v := range idxFieldKey {
		require.Equal(t, ifk[k], v)
	}
}
