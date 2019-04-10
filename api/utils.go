package api

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// RunAsync runs the specified function asynchronously.
func RunAsync(f func() error) <-chan error {
	resp := make(chan error, 1)
	go func() {
		err := f()
		resp <- err
		close(resp)
	}()
	return resp
}

// HashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
// This gives context to the signed message and prevents signing of transactions.
func HashMessage(message string) []byte {
	data := []byte(message)
	if strings.HasPrefix(message, "0x") {
		if value, err := hex.DecodeString(message[2:]); err == nil {
			data = value
		}
	}

	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
