package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const (
	aesNonceLength = 12
)

func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func validateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if expectedSize > 3 && containsOnlyZeros(k) {
		return false
	}
	return true
}

func generateSecureRandomData(length int) ([]byte, error) {
	res := make([]byte, length)

	_, err := rand.Read(res)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(res, length) {
		return nil, errors.New("crypto/rand failed to generate secure random data")
	}
	return res, nil
}

func EncryptSymmetric(key []byte, plaintext []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	salt, err := generateSecureRandomData(aesNonceLength)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	encrypted, err := aesgcm.Seal(nil, salt, plaintext, nil), nil
	if err != nil {
		return nil, err
	}

	return append(encrypted, salt...), nil
}

func DecryptSymmetric(key []byte, cyphertext []byte) ([]byte, error) {
	// symmetric messages are expected to contain the 12-byte nonce at the end of the payload
	if len(cyphertext) < aesNonceLength {
		return nil, errors.New("missing salt or invalid payload in symmetric message")
	}
	salt := cyphertext[len(cyphertext)-aesNonceLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	decrypted, err := aesgcm.Open(nil, salt, cyphertext[:len(cyphertext)-aesNonceLength], nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}
