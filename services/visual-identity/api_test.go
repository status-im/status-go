package visualidentity

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestAPI(t *testing.T) *API {
	api := NewAPI()

	alphabet, err := LoadAlphabet()
	require.NoError(t, err)

	api.emojisAlphabet = alphabet
	return api
}

func TestEmojiHashOf(t *testing.T) {
	api := setupTestAPI(t)

	checker := func(pubkey string, expected *[](string)) {
		emojihash, err := api.EmojiHashOf(pubkey)
		require.NoError(t, err)
		if !reflect.DeepEqual(emojihash, *expected) {
			t.Fatalf("invalid emojihash %v != %v", emojihash, *expected)
		}
	}

	checker("0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
		&[](string){"🧒🏽", "🤰🏽", "👄", "🧵", "🐍", "👏🏻", "🧏‍♀️", "👏🏾", "🤥", "🧑🏽‍🤝‍🧑🏼", "🧑🏽‍🍳", "🥅", "🎣", "👶🏽"})

	checker("0x0400000000000000000000000000000000000000000000000000000000000000014218F20AE6C646B363DB68605822FB14264CA8D2587FDD6FBC750D587E76A7EE",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀"})

	checker("0x04000000000000000000000000000000000000000010000000000000000000000033600332D373318ECC2F212A30A5750D2EAC827B6A32B33D326CCF369B12B1BE",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", (*api.emojisAlphabet)[1]})

	checker("0x040000000000000000000000000000000000000000200000000000000000000000353050BFE33B724E60A0C600FBA565A9B62217B1BD35BF9848F2AB847C598B30",
		&[](string){"😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", "😀", (*api.emojisAlphabet)[2]})
}

func TestColorHashOf(t *testing.T) {
	api := NewAPI()

	checker := func(pubkey string, expected *[][](int)) {
		colorhash, err := api.ColorHashOf(pubkey)
		require.NoError(t, err)
		if !reflect.DeepEqual(colorhash, *expected) {
			t.Fatalf("invalid emojihash %v != %v", colorhash, *expected)
		}
	}

	checker("0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
		&[][]int{{3, 30}, {2, 10}, {5, 5}, {3, 14}, {5, 4}, {4, 19}, {3, 16}, {4, 0}, {5, 28}, {4, 13}, {4, 15}})
}

func TestHashesOfInvalidKey(t *testing.T) {
	api := setupTestAPI(t)

	checker := func(pubkey string) {
		_, err := api.EmojiHashOf(pubkey)
		require.Error(t, err)
		_, err = api.ColorHashOf(pubkey)
		require.Error(t, err)
	}
	checker("abc")
	checker("0x01")
	checker("0x01e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
	checker("0x04425da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
}
