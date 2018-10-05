package hexutils

import (
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
)

// HexToBytes convert a hex string to a byte sequence.
// The hex string can have spaces between bytes.
func HexToBytes(s string) []byte {
	s = regexp.MustCompile(" ").ReplaceAllString(s, "")
	b := make([]byte, hex.DecodedLen(len(s)))
	_, err := hex.Decode(b, []byte(s))
	if err != nil {
		log.Fatal(err)
	}

	return b[:]
}

// BytesToHexWithSpaces returns an hex string of b adding spaces between bytes.
func BytesToHexWithSpaces(b []byte) string {
	return fmt.Sprintf("% X", b)
}

// BytesToHex returns an hex string of b.
func BytesToHex(b []byte) string {
	return fmt.Sprintf("%X", b)
}
