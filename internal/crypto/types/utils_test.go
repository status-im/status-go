package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsHexCharacter(t *testing.T) {
	tests := []struct {
		character string
		want      bool
	}{
		{"0", true},
		{"1", true},
		{"2", true},
		{"3", true},
		{"4", true},
		{"5", true},
		{"6", true},
		{"7", true},
		{"8", true},
		{"9", true},
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", true},
		{"e", true},
		{"f", true},
		{"A", true},
		{"B", true},
		{"C", true},
		{"D", true},
		{"E", true},
		{"F", true},
		{"-", false},
		{".", false},
		{"_", false},
		{" ", false},
		{"x", false},
		{"X", false},
		{"h", false},
		{"H", false},
	}

	for _, test := range tests {
		t.Run(test.character, func(t *testing.T) {
			got := isHexCharacter([]byte(test.character)[0])
			require.Equal(t, test.want, got)
		})
	}
}

func TestIsHexAddress(t *testing.T) {
	tests := []struct {
		address      string
		has0x        bool
		isHex        bool
		isHexAddress bool
	}{
		{"0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa", true, false, true},
		{"aAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa", false, true, true},
		{"0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaA", true, false, false},
		{"aAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaA", false, false, false},
		{"0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaa", true, false, false},
		{"aAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaa", false, true, false},
	}

	for _, test := range tests {
		t.Run(test.address, func(t *testing.T) {
			gotHas0x := has0xPrefix(test.address)
			require.Equal(t, test.has0x, gotHas0x)

			gotIsHex := isHex(test.address)
			require.Equal(t, test.isHex, gotIsHex)

			gotIsHexAddress := IsHexAddress(test.address)
			require.Equal(t, test.isHexAddress, gotIsHexAddress)
		})
	}
}
