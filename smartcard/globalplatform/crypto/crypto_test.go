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

func TestAppendDESPadding(t *testing.T) {
	data := hexutils.HexToBytes("AABB")
	result := appendDESPadding(data)
	expected := "AABB800000000000"
	assert.Equal(t, expected, hexutils.BytesToHex(result))
}

func TestVerifyCryptogram(t *testing.T) {
	encKey := hexutils.HexToBytes("16B5867FF50BE7239C2BF1245B83A362")
	hostChallenge := hexutils.HexToBytes("32da078d7aac1cff")
	cardChallenge := hexutils.HexToBytes("007284f64a7d6465")
	cardCryptogram := hexutils.HexToBytes("05c4bb8a86014e22")

	result, err := VerifyCryptogram(encKey, hostChallenge, cardChallenge, cardCryptogram)
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestMac3des(t *testing.T) {
	key := hexutils.HexToBytes("16B5867FF50BE7239C2BF1245B83A362")
	data := hexutils.HexToBytes("32DA078D7AAC1CFF007284F64A7D64658000000000000000")
	result, err := mac3des(key, data, NullBytes8)
	assert.NoError(t, err)

	expected := "05C4BB8A86014E22"
	assert.Equal(t, expected, hexutils.BytesToHex(result))
}

func TestMacFull3DES(t *testing.T) {
	key := hexutils.HexToBytes("5b02e75ad63190aece0622936f11abab")
	data := hexutils.HexToBytes("8482010010810b098a8fbb88da")
	result, err := MacFull3DES(key, data, NullBytes8)
	assert.NoError(t, err)
	expected := "5271D7174A5A166A"
	assert.Equal(t, expected, hexutils.BytesToHex(result))
}
