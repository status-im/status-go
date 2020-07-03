package push_notification_server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"github.com/status-im/status-go/eth-node/crypto"
	"golang.org/x/crypto/sha3"
	"io"
)

func hashPublicKey(pk *ecdsa.PublicKey) []byte {
	return shake256(crypto.CompressPubkey(pk))
}

func decrypt(cyphertext []byte, key []byte) ([]byte, error) {
	if len(cyphertext) < nonceLength {
		return nil, ErrInvalidCiphertextLength
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := cyphertext[:nonceLength]
	return gcm.Open(nil, nonce, cyphertext[nonceLength:], nil)
}

func encrypt(plaintext []byte, key []byte, reader io.Reader) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func shake256(buf []byte) []byte {
	h := make([]byte, 64)
	sha3.ShakeSum256(h, buf)
	return h
}
