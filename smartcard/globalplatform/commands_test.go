package globalplatform

import (
	"testing"

	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestNewCommandSelect(t *testing.T) {
	aid := []byte{}
	cmd := NewCommandSelect(aid)

	assert.Equal(t, uint8(0x00), cmd.Cla)
	assert.Equal(t, uint8(0xA4), cmd.Ins)
	assert.Equal(t, uint8(0x04), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
}

func TestNewCommandInitializeUpdate(t *testing.T) {
	challenge := hexutils.HexToBytes("010203")
	cmd := NewCommandInitializeUpdate(challenge)

	assert.Equal(t, uint8(0x80), cmd.Cla)
	assert.Equal(t, uint8(0x50), cmd.Ins)
	assert.Equal(t, uint8(0x00), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
	assert.Equal(t, challenge, cmd.Data)
}

func TestCalculateHostCryptogram(t *testing.T) {
	encKey := hexutils.HexToBytes("0EF72A1065236DD6CAC718D5E3F379A4")
	cardChallenge := hexutils.HexToBytes("0076a6c0d55e9535")
	hostChallenge := hexutils.HexToBytes("266195e638da1b95")

	result, err := calculateHostCryptogram(encKey, cardChallenge, hostChallenge)
	assert.NoError(t, err)

	expected := "45A5F48DAE68203C"
	assert.Equal(t, expected, hexutils.BytesToHex(result))
}

func TestNewCommandExternalAuthenticate(t *testing.T) {
	encKey := hexutils.HexToBytes("8D289AFE0AB9C45B1C76DEEA182966F4")
	cardChallenge := hexutils.HexToBytes("000f3fd65d4d6e45")
	hostChallenge := hexutils.HexToBytes("cf307b6719bf224d")

	cmd, err := NewCommandExternalAuthenticate(encKey, cardChallenge, hostChallenge)
	assert.NoError(t, err)

	expected := "84 82 01 00 08 77 02 AC 6C E4 6A 47 F0"
	raw, err := cmd.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, expected, hexutils.BytesToHexWithSpaces(raw))
}

func TestNewCommandDelete(t *testing.T) {
	aid := hexutils.HexToBytes("0102030405")
	cmd := NewCommandDelete(aid)
	assert.Equal(t, uint8(0x80), cmd.Cla)
	assert.Equal(t, uint8(0xE4), cmd.Ins)
	assert.Equal(t, uint8(0x00), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
	assert.Equal(t, aid, cmd.Data)
}
