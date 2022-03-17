package emojihash

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/identity"
)

func TestGenerateFor(t *testing.T) {
	checker := func(pubkey string, expected *[](string)) {
		emojihash, err := GenerateFor(pubkey)
		require.NoError(t, err)
		if !reflect.DeepEqual(emojihash, *expected) {
			t.Fatalf("invalid emojihash %v != %v", emojihash, *expected)
		}
	}

	checker("0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
		&[](string){"👦🏽", "🦹🏻", "👶🏿", "🛁", "🌁", "🙌🏻", "🙇🏽‍♂️", "🙌🏾", "🤥", "🐛", "👩🏽‍🔧", "🔧", "⚙️", "🧒🏽"})

	checker("0x0400000000000000000000000000000000000000000000000000000000000000014218F20AE6C646B363DB68605822FB14264CA8D2587FDD6FBC750D587E76A7EE",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀"})

	checker("0x04000000000000000000000000000000000000000010000000000000000000000033600332D373318ECC2F212A30A5750D2EAC827B6A32B33D326CCF369B12B1BE",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", (emojisAlphabet)[1]})

	checker("0x040000000000000000000000000000000000000000200000000000000000000000353050BFE33B724E60A0C600FBA565A9B62217B1BD35BF9848F2AB847C598B30",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", (emojisAlphabet)[2]})
}

func TestEmojiHashOfInvalidKey(t *testing.T) {
	checker := func(pubkey string) {
		_, err := GenerateFor(pubkey)
		require.Error(t, err)
	}
	checker("abc")
	checker("0x01")
	checker("0x01e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
	checker("0x04425da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
}

func TestToEmojiHash(t *testing.T) {
	alphabet := [](string){"😇", "🤐", "🥵", "🙊", "🤌"}

	checker := func(valueStr string, hashLen int, expected *[](string)) {
		value := identity.ToBigInt(t, valueStr)
		res, err := toEmojiHash(value, hashLen, &alphabet)
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
	val := identity.ToBigInt(t, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF") // 20 bytes
	_, err := toEmojiHash(val, 14, &alphabet)
	require.NoError(t, err)

	alphabet = make([](string), 2757-1)
	_, err = toEmojiHash(val, 14, &alphabet)
	require.Error(t, err)
}
