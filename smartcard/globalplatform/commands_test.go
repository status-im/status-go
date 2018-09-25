package globalplatform

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

func TestCommandSelect(t *testing.T) {
	aid := []byte{}
	cmd := NewCommandSelect(aid)

	assert.Equal(t, uint8(0x00), cmd.Cla)
	assert.Equal(t, uint8(0xA4), cmd.Ins)
	assert.Equal(t, uint8(0x04), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
}

func TestCommandInitializeUpdate(t *testing.T) {
	challenge := hexToBytes("010203")
	cmd := NewCommandInitializeUpdate(challenge)

	assert.Equal(t, uint8(0x80), cmd.Cla)
	assert.Equal(t, uint8(0x50), cmd.Ins)
	assert.Equal(t, uint8(0x00), cmd.P1)
	assert.Equal(t, uint8(0x00), cmd.P2)
	assert.Equal(t, challenge, cmd.Data)
}
