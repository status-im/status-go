package crypto

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
)

var (
	DerivationPurposeEnc = []byte{0x01, 0x82}
	DerivationPurposeMac = []byte{0x01, 0x01}
	NullBytes8           = []byte{0, 0, 0, 0, 0, 0, 0, 0}
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

	mode := cipher.NewCBCEncrypter(block, NullBytes8)
	mode.CryptBlocks(ciphertext, derivation)

	return ciphertext, nil
}

func VerifyCryptogram(encKey, hostChallenge, cardChallenge, cardCryptogram []byte) (bool, error) {
	data := make([]byte, 0)
	data = append(data, hostChallenge...)
	data = append(data, cardChallenge...)
	paddedData := appendDESPadding(data)
	calculated, err := mac3des(encKey, paddedData, NullBytes8)
	if err != nil {
		return false, err
	}

	return bytes.Equal(calculated, cardCryptogram), nil
}

func MacFull3DES(key, data, iv []byte) ([]byte, error) {
	data = appendDESPadding(data)

	desBlock, err := des.NewCipher(resizeKey8(key))
	if err != nil {
		return nil, err
	}

	des3Block, err := des.NewTripleDESCipher(resizeKey24(key))
	if err != nil {
		return nil, err
	}

	des3IV := iv

	if len(data) > 8 {
		length := len(data) - 8
		tmp := make([]byte, length)
		mode := cipher.NewCBCEncrypter(desBlock, iv)
		mode.CryptBlocks(tmp, data[:length])
		des3IV = tmp
	}

	ciphertext := make([]byte, 8)

	mode := cipher.NewCBCEncrypter(des3Block, des3IV)
	mode.CryptBlocks(ciphertext, data[len(data)-8:])

	return ciphertext, nil
}

func EncryptICV(macKey, mac []byte) ([]byte, error) {
	block, err := des.NewCipher(resizeKey8(macKey))
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, 16)
	mode := cipher.NewCBCEncrypter(block, NullBytes8)
	mode.CryptBlocks(ciphertext, mac)

	return ciphertext, nil
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

func resizeKey8(key []byte) []byte {
	return key[:8]
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
