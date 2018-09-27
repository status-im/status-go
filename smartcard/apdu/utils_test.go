package apdu

import (
	"testing"

	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestFindTag(t *testing.T) {
	var (
		tagData []byte
		err     error
	)

	data := hexutils.HexToBytes("C1 02 BB CC C2 04 C3 02 11 22 C3 02 88 99")

	tagData, err = FindTag(data, uint8(0xC1))
	assert.NoError(t, err)
	assert.Equal(t, "BB CC", hexutils.BytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC2))
	assert.NoError(t, err)
	assert.Equal(t, "C3 02 11 22", hexutils.BytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC3))
	assert.NoError(t, err)
	assert.Equal(t, "88 99", hexutils.BytesToHexWithSpaces(tagData))

	tagData, err = FindTag(data, uint8(0xC2), uint8(0xC3))
	assert.NoError(t, err)
	assert.Equal(t, "11 22", hexutils.BytesToHexWithSpaces(tagData))

	// tag not found
	data = hexutils.HexToBytes("C1 00")
	_, err = FindTag(data, uint8(0xC2))
	assert.Equal(t, &ErrTagNotFound{uint8(0xC2)}, err)

	// sub-tag not found
	data = hexutils.HexToBytes("C1 02 C2 00")
	_, err = FindTag(data, uint8(0xC1), uint8(0xC3))
	assert.Equal(t, &ErrTagNotFound{uint8(0xC3)}, err)
}
