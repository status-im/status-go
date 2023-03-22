package colorhash

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/identity"
)

func TestGenerateFor(t *testing.T) {
	checker := func(pubkey string, expected *multiaccounts.ColorHash) {
		colorhash, err := GenerateFor(pubkey)
		require.NoError(t, err)
		if !reflect.DeepEqual(colorhash, *expected) {
			t.Fatalf("invalid emojihash %v != %v", colorhash, *expected)
		}
	}

	checker("0x04e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8",
		&multiaccounts.ColorHash{{3, 30}, {2, 10}, {5, 5}, {3, 14}, {5, 4}, {4, 19}, {3, 16}, {4, 0}, {5, 28}, {4, 13}, {4, 15}})
}

func TestColorHashOfInvalidKey(t *testing.T) {
	checker := func(pubkey string) {
		_, err := GenerateFor(pubkey)
		require.Error(t, err)
	}
	checker("abc")
	checker("0x01")
	checker("0x01e25da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
	checker("0x04425da6994ea2dc4ac70727e07eca153ae92bf7609db7befb7ebdceaad348f4fc55bbe90abf9501176301db5aa103fc0eb3bc3750272a26c424a10887db2a7ea8")
}

func TestColorHash(t *testing.T) {
	alphabet := makeColorHashAlphabet(4, 4)

	checker := func(valueStr string, expected *multiaccounts.ColorHash) {
		value := identity.ToBigInt(t, valueStr)
		res := toColorHash(value, &alphabet, 4)
		if !reflect.DeepEqual(res, *expected) {
			t.Fatalf("invalid colorhash conversion %v != %v", res, *expected)
		}
	}

	checker("0x0", &multiaccounts.ColorHash{{1, 0}})
	checker("0x1", &multiaccounts.ColorHash{{1, 1}})
	checker("0x4", &multiaccounts.ColorHash{{2, 0}})
	checker("0xF", &multiaccounts.ColorHash{{4, 3}})

	// oops, collision
	checker("0xFF", &multiaccounts.ColorHash{{4, 3}, {4, 0}})
	checker("0xFC", &multiaccounts.ColorHash{{4, 3}, {4, 0}})

	checker("0xFFFF", &multiaccounts.ColorHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}})
}
