package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

// Decrypt implements GCM decryption. Key and nonce is in hex representation
func Decrypt(keyHex, nonceHex string, cipherText []byte) ([]byte, error) {
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

	plaintext, err := aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
