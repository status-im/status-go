package identity

import (
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToBigBase(t *testing.T) {
	checker := func(value *big.Int, base uint64, expected *[](uint64)) {
		res := ToBigBase(value, base)
		if !reflect.DeepEqual(res, *expected) {
			t.Fatalf("invalid big base conversion %v != %v", res, *expected)
		}
	}

	lengthChecker := func(value *big.Int, base, expectedLength uint64) {
		res := ToBigBase(value, base)
		if len(res) != int(expectedLength) {
			t.Fatalf("invalid big base conversion %d != %d", len(res), expectedLength)
		}
	}

	checker(new(big.Int).SetUint64(15), 16, &[](uint64){15})
	checker(new(big.Int).SetUint64(495), 16, &[](uint64){1, 14, 15})
	checker(new(big.Int).SetUint64(495), 30, &[](uint64){16, 15})
	checker(new(big.Int).SetUint64(495), 1024, &[](uint64){495})
	checker(new(big.Int).SetUint64(2048), 1024, &[](uint64){2, 0})

	base := uint64(math.Pow(2, 7*4))
	checker(ToBigInt(t, "0xFFFFFFFFFFFFFF"), base, &[](uint64){base - 1, base - 1})

	val := ToBigInt(t, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	lengthChecker(val, 2757, 14)
	lengthChecker(val, 2756, 15)
}

func TestSlices(t *testing.T) {
	checker := func(compressedKey, charsCutoffA, emojiHash, colorHash, charsCutoffB string) {
		slices, err := Slices(ToBigInt(t, compressedKey).Bytes())
		require.NoError(t, err)

		sliceChecker := func(idx int, value *big.Int) {
			if !reflect.DeepEqual(slices[idx], value.Bytes()) {
				t.Fatalf("invalid slice (%d) %v != %v", idx, slices[idx], value.Bytes())
			}
		}

		sliceChecker(0, ToBigInt(t, charsCutoffA))
		sliceChecker(1, ToBigInt(t, emojiHash))
		sliceChecker(2, ToBigInt(t, colorHash))
		sliceChecker(3, ToBigInt(t, charsCutoffB))
	}

	checker("0x03086138b210f21d41c757ae8a5d2a4cb29c1350f7389517608378ebd9efcf4a55", "0x030", "0x86138b210f21d41c757ae8a5d2a4cb29c1350f73", "0x89517608378ebd9efcf4", "0xa55")
	checker("0x020000000000000000000000000000000000000000100000000000000000000000", "0x020", "0x0000000000000000000000000000000000000001", "0x00000000000000000000", "0x000")
}

func TestSlicesInvalid(t *testing.T) {
	_, err := Slices(ToBigInt(t, "0x01").Bytes())
	require.Error(t, err)
}
