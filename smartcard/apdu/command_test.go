package apdu

import (
	"testing"

	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestNewCommand(t *testing.T) {
	var cla uint8 = 0x80
	var ins uint8 = 0x50
	var p1 uint8 = 1
	var p2 uint8 = 2
	data := hexutils.HexToBytes("84762336c5187fe8")

	cmd := NewCommand(cla, ins, p1, p2, data)

	expected := "80 50 01 02 08 84 76 23 36 C5 18 7F E8"
	result, err := cmd.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, expected, hexutils.BytesToHexWithSpaces(result))

	cmd.SetLE(uint8(0x77))
	expected = "80 50 01 02 08 84 76 23 36 C5 18 7F E8 77"
	result, err = cmd.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, expected, hexutils.BytesToHexWithSpaces(result))
}
