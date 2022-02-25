package visualidentity

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
	checker(toBigInt(t, "0xFFFFFFFFFFFFFF"), base, &[](uint64){base - 1, base - 1})

	val := toBigInt(t, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	lengthChecker(val, 2757, 14)
	lengthChecker(val, 2756, 15)
}

func TestToEmojiHash(t *testing.T) {
	alphabet := [](string){"😇", "🤐", "🥵", "🙊", "🤌"}

	checker := func(valueStr string, hashLen int, expected *[](string)) {
		value := toBigInt(t, valueStr)
		res, err := ToEmojiHash(value, hashLen, &alphabet)
		require.NoError(t, err)
		if !reflect.DeepEqual(res, *expected) {
			t.Fatalf("invalid emojihash conversion %v != %v", res, *expected)
		}
	}

	checker("777", 5, &[](string){"🤐", "🤐", "🤐", "😇", "🥵"})
	checker("777", 0, &[](string){"🤐", "🤐", "🤐", "😇", "🥵"})
	checker("777", 10, &[](string){"😇", "😇", "😇", "😇", "😇", "🤐", "🤐", "🤐", "😇", "🥵"})

	// 20bytes of data described by 14 emojis requires at least 2757 length alphabet
	alphabet = make([](string), 2757)
	val := toBigInt(t, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") // 20 bytes
	_, err := ToEmojiHash(val, 14, &alphabet)
	require.NoError(t, err)

	alphabet = make([](string), 2757-1)
	_, err = ToEmojiHash(val, 14, &alphabet)
	require.Error(t, err)
}

func TestSlices(t *testing.T) {
	checker := func(compressedKey, charsCutoffA, emojiHash, colorHash, charsCutoffB string) {
		slices, err := Slices(toBigInt(t, compressedKey).Bytes())
		require.NoError(t, err)

		sliceChecker := func(idx int, value *big.Int) {
			if !reflect.DeepEqual(slices[idx], value.Bytes()) {
				t.Fatalf("invalid slice (%d) %v != %v", idx, slices[idx], value.Bytes())
			}
		}

		sliceChecker(0, toBigInt(t, charsCutoffA))
		sliceChecker(1, toBigInt(t, emojiHash))
		sliceChecker(2, toBigInt(t, colorHash))
		sliceChecker(3, toBigInt(t, charsCutoffB))
	}

	checker("0x03086138b210f21d41c757ae8a5d2a4cb29c1350f7389517608378ebd9efcf4a55", "0x030", "0x86138b210f21d41c757ae8a5d2a4cb29c1350f73", "0x89517608378ebd9efcf4", "0xa55")
	checker("0x020000000000000000000000000000000000000000100000000000000000000000", "0x020", "0x0000000000000000000000000000000000000001", "0x00000000000000000000", "0x000")
}

func TestSlicesInvalid(t *testing.T) {
	_, err := Slices(toBigInt(t, "0x01").Bytes())
	require.Error(t, err)
}

func TestColorHash(t *testing.T) {
	alphabet := MakeColorHashAlphabet(4, 4)

	checker := func(valueStr string, expected *[][](int)) {
		value := toBigInt(t, valueStr)
		res := ToColorHash(value, &alphabet, 4)
		if !reflect.DeepEqual(res, *expected) {
			t.Fatalf("invalid colorhash conversion %v != %v", res, *expected)
		}
	}

	checker("0x0", &[][]int{{1, 0}})
	checker("0x1", &[][]int{{1, 1}})
	checker("0x4", &[][]int{{2, 0}})
	checker("0xF", &[][]int{{4, 3}})

	// oops, collision
	checker("0xFF", &[][]int{{4, 3}, {4, 0}})
	checker("0xFC", &[][]int{{4, 3}, {4, 0}})

	checker("0xFFFF", &[][]int{{4, 3}, {4, 0}, {4, 3}, {4, 0}})
}

func toBigInt(t *testing.T, str string) *big.Int {
	res, ok := new(big.Int).SetString(str, 0)
	if !ok {
		t.Errorf("invalid conversion to int from %s", str)
	}
	return res
}
