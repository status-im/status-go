package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var tcs = []int{4, 16, 64, 256, 1024}

func runeInSlice(a rune, list []rune) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func validString(s string, runes []rune) bool {
	for _, r := range s {
		if !runeInSlice(r, runes) {
			return false
		}
	}
	return true
}

func TestRandomAlphabeticalString(t *testing.T) {
	for _, n := range tcs {
		s, err := RandomAlphabeticalString(n)
		require.NoError(t, err)
		require.Len(t, s, n)

		require.True(t, validString(s, letterRunes))
	}
}

func TestRandomAlphanumericString(t *testing.T) {
	for _, n := range tcs {
		s, err := RandomAlphanumericString(n)
		require.NoError(t, err)
		require.Len(t, s, n)

		require.True(t, validString(s, alphanumericRunes))
	}
}
