package identicon

import (
	"crypto/md5" // nolint: gosec
	"reflect"
	"testing"
)

var GeneratedPattern = []byte{
	129, 201, 130, 201, 129,
	56, 157, 244, 157, 56,
	232, 187, 241, 187, 232,
	183, 61, 165, 61, 183,
	149, 108, 179, 108, 149,
}

var GeneratedBitmap = []byte{
	0, 0, 1, 0, 0,
	1, 0, 1, 0, 1,
	1, 0, 0, 0, 1,
	0, 0, 0, 0, 0,
	0, 1, 0, 1, 0,
}

var GeneratedHash = md5.Sum([]byte("Culona")) // nolint: gosec

func TestItGeneratesAPatternFromAListOfBytes(t *testing.T) {
	pattern := generatePatternFromHash(GeneratedHash)

	if !reflect.DeepEqual(GeneratedPattern, pattern) {
		t.Fatal("Failing asserting equality of pattern.")
	}
}

func TestItGeneratesABitMapFromPattern(t *testing.T) {
	bitmap := convertPatternToBinarySwitch(GeneratedPattern)

	if !reflect.DeepEqual(GeneratedBitmap, bitmap) {
		t.Fatal("Failing asserting equality of bitmap.")
	}
}

func TestItGeneratesAnIdenticonFromString(t *testing.T) {
	identicon := generate("Culona")

	if !reflect.DeepEqual(GeneratedBitmap, identicon.bitmap) {
		t.Fatal("Failing asserting that the identicon has a valid  bitmap.")
	}
}
