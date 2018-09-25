package apdu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindTag(t *testing.T) {
	var (
		tagData []byte
		err     error
	)

	data := hexToBytes("C1 02 BB CC C2 04 C3 02 11 22 C3 02 88 99")

	tagData, err = FindTag(data, uint8(0xC1))
	assert.NoError(t, err)
	assert.Equal(t, "BB CC", bytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC2))
	assert.NoError(t, err)
	assert.Equal(t, "C3 02 11 22", bytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC3))
	assert.NoError(t, err)
	assert.Equal(t, "88 99", bytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC2), uint8(0xC3))
	assert.NoError(t, err)
	assert.Equal(t, "11 22", bytesToHexWithSpaces(tagData))

	// tag not found
	data = hexToBytes("C1 00")
	_, err = FindTag(data, uint8(0xC2))
	assert.Equal(t, &ErrTagNotFound{uint8(0xC2)}, err)

	// sub-tag not found
	data = hexToBytes("C1 02 C2 00")
	_, err = FindTag(data, uint8(0xC1), uint8(0xC3))
	assert.Equal(t, &ErrTagNotFound{uint8(0xC3)}, err)
}
