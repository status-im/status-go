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
	raw, err := cmd.Serialize()
	assert.NoError(t, err)

	expected := "00 A4 04 00 00"
	assert.Equal(t, expected, bytesToHexWithSpaces(raw))
}
