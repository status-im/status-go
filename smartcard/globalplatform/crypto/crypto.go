package crypto

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
)

var (
	// DerivationPurposeEnc defines 2 bytes used when deriving a encoding key.
	DerivationPurposeEnc = []byte{0x01, 0x82}
	// DerivationPurposeMac defines 2 bytes used when deriving a mac key.
	DerivationPurposeMac = []byte{0x01, 0x01}
	// NullBytes8 defined a slice of 8 zero bytes mostrly used as IV in cryptographic functions.
	NullBytes8 = []byte{0, 0, 0, 0, 0, 0, 0, 0}
)

// DeriveKey derives a key from the current cardKey using the sequence number receive from the card and the purpose (ENC/MAC).
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

// VerifyCryptogram verifies the cryptogram sends from the card to ensure that card and client are using the same keys to communicate.
func VerifyCryptogram(encKey, hostChallenge, cardChallenge, cardCryptogram []byte) (bool, error) {
	data := make([]byte, 0)
	data = append(data, hostChallenge...)
	data = append(data, cardChallenge...)
	paddedData := AppendDESPadding(data)
	calculated, err := Mac3DES(encKey, paddedData, NullBytes8)
	if err != nil {
		return false, err
	}

	return bytes.Equal(calculated, cardCryptogram), nil
}

// MacFull3DES generates a full triple DES mac.
func MacFull3DES(key, data, iv []byte) ([]byte, error) {
	data = AppendDESPadding(data)

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
		des3IV = tmp[length-8:]
	}

	ciphertext := make([]byte, 8)

	mode := cipher.NewCBCEncrypter(des3Block, des3IV)
	mode.CryptBlocks(ciphertext, data[len(data)-8:])

	return ciphertext, nil
}

// EncryptICV encrypts an ICV with the specified macKey.
// The ICV is usually the mac of the previous command sent in the current session.
func EncryptICV(macKey, icv []byte) ([]byte, error) {
	block, err := des.NewCipher(resizeKey8(macKey))
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, 8)
	mode := cipher.NewCBCEncrypter(block, NullBytes8)
	mode.CryptBlocks(ciphertext, icv)

	return ciphertext, nil
}

// Mac3DES generates the triple DES mac of data using the specified key and icv.
func Mac3DES(key, data, iv []byte) ([]byte, error) {
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

// AppendDESPadding appends an 0x80 bytes to data and other zero bytes to make the result length multiple of 8.
func AppendDESPadding(data []byte) []byte {
	length := len(data) + 1
	for ; length%8 != 0; length++ {
	}

	newData := make([]byte, length)
	copy(newData, data)
	copy(newData[len(data):], []byte{0x80})

	return newData
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
