package apdu

import (
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func hexToBytes(s string) []byte {
	b := make([]byte, hex.DecodedLen(len(s)))
	_, err := hex.Decode(b, []byte(s))
	if err != nil {
		log.Fatal(err)
	}

	return b[:]
}

func bytesToHexWithSpaces(b []byte) string {
	return fmt.Sprintf("% X", b)
}

func bytesToHex(b []byte) string {
	return fmt.Sprintf("%X", b)
}

func TestNewCommand(t *testing.T) {
	var cla uint8 = 0x80
	var ins uint8 = 0x50
	var p1 uint8 = 1
	var p2 uint8 = 2
	data := hexToBytes("84762336c5187fe8")

	cmd := NewCommand(cla, ins, p1, p2, data)
	expected := "80 50 01 02 08 84 76 23 36 C5 18 7F E8 00"

	result, err := cmd.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, expected, bytesToHexWithSpaces(result))
}
