package hexutils

import (
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
)

func HexToBytes(s string) []byte {
	s = regexp.MustCompile(" ").ReplaceAllString(s, "")
	b := make([]byte, hex.DecodedLen(len(s)))
	_, err := hex.Decode(b, []byte(s))
	if err != nil {
		log.Fatal(err)
	}

	return b[:]
}

func BytesToHexWithSpaces(b []byte) string {
	return fmt.Sprintf("% X", b)
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("%X", b)
}
