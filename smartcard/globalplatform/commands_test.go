package globalplatform

import (
	"testing"

	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestCommandSelect(t *testing.T) {
	aid := []byte{}
	cmd := NewCommandSelect(aid)

	assert.Equal(t, uint8(0x00), cmd.Cla)
	assert.Equal(t, uint8(0xA4), cmd.Ins)
	assert.Equal(t, uint8(0x04), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
}

func TestCommandInitializeUpdate(t *testing.T) {
	challenge := hexutils.HexToBytes("010203")
	cmd := NewCommandInitializeUpdate(challenge)

	assert.Equal(t, uint8(0x80), cmd.Cla)
	assert.Equal(t, uint8(0x50), cmd.Ins)
	assert.Equal(t, uint8(0x00), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
	assert.Equal(t, challenge, cmd.Data)
}
