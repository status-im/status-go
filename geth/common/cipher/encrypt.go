package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
)

// Encrypt implements GCM encryption. Key and nonce are in hex representation.
func Encrypt(keyHex, nonceHex string, plaintext []byte) ([]byte, error) {
	// The key argument should be the AES key, either 16 or 32 bytes (32, 64 bytes in hex representation)
	// to select AES-128 or AES-256.
	if len(keyHex) != 32 && len(keyHex) != 64 {
		return nil, errors.New("invalid key length")
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, err
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(nil, nonce, plaintext, nil), nil
}
