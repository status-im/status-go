package crypto

import (
	"testing"

	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestDeriveKey(t *testing.T) {
	cardKey := hexutils.HexToBytes("404142434445464748494a4b4c4d4e4f")
	seq := hexutils.HexToBytes("0065")

	encKey, err := DeriveKey(cardKey, seq, DerivationPurposeEnc)
	assert.NoError(t, err)

	expectedEncKey := "85E72AAF47874218A202BF5EF891DD21"
	assert.Equal(t, expectedEncKey, hexutils.BytesToHex(encKey))
}

func TestResizeKey24(t *testing.T) {
	key := hexutils.HexToBytes("404142434445464748494a4b4c4d4e4f")
	resized := resizeKey24(key)
	expected := "404142434445464748494A4B4C4D4E4F4041424344454647"
	assert.Equal(t, expected, hexutils.BytesToHex(resized))
}
