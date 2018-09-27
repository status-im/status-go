package crypto

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
)

var (
	DerivationPurposeEnc = []byte{0x01, 0x82}
	nullBytes8           = []byte{0, 0, 0, 0, 0, 0, 0, 0}
)

func DeriveKey(cardKey []byte, seq []byte, purpose []byte) ([]byte, error) {
	key24 := resizeKey24(cardKey)

	derivation := make([]byte, 16)
	copy(derivation, purpose[:2])
	copy(derivation[2:], seq[:2])

	block, err := des.NewTripleDESCipher(key24)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, 16)

	mode := cipher.NewCBCEncrypter(block, nullBytes8)
	mode.CryptBlocks(ciphertext, derivation)

	return ciphertext, nil
}

func VerifyCryptogram(encKey, hostChallenge, cardChallenge, cardCryptogram []byte) (bool, error) {
	data := make([]byte, 0)
	data = append(data, hostChallenge...)
	data = append(data, cardChallenge...)
	paddedData := appendDESPadding(data)
	calculated, err := mac3des(encKey, paddedData, nullBytes8)
	if err != nil {
		return false, err
	}

	return bytes.Equal(calculated, cardCryptogram), nil
}

func mac3des(key, data, iv []byte) ([]byte, error) {
	key24 := resizeKey24(key)

	block, err := des.NewTripleDESCipher(key24)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, 24)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, data)

	return ciphertext[16:], nil
}

func resizeKey24(key []byte) []byte {
	data := make([]byte, 24)
	copy(data, key[0:16])
	copy(data[16:], key[0:8])

	return data
}

func appendDESPadding(data []byte) []byte {
	length := len(data) + 1
	for ; length%8 != 0; length++ {
	}

	newData := make([]byte, length)
	copy(newData, data)
	copy(newData[len(data):], []byte{0x80})

	return newData
}
