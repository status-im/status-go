package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

// CipherExt defines encrypted files.
const CipherExt = ".cr"

// Decrypt implements GCM decryption. Key and nonce are in hex representation.
func Decrypt(keyHex, nonceHex string, cipherText []byte) (plaintext []byte, err error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	plaintext, err = aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return
	}

	return plaintext, nil
}
