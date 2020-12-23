package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"errors"
	"io"

	"golang.org/x/crypto/sha3"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

const nonceLength = 12

var ErrInvalidCiphertextLength = errors.New("invalid cyphertext length")

func HashPublicKey(pk *ecdsa.PublicKey) []byte {
	return Shake256(crypto.CompressPubkey(pk))
}

func Decrypt(cyphertext []byte, key []byte) ([]byte, error) {
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

func Encrypt(plaintext []byte, key []byte, reader io.Reader) ([]byte, error) {
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

func Shake256(buf []byte) []byte {
	h := make([]byte, 64)
	sha3.ShakeSum256(h, buf)
	return h
}

// IsPubKeyEqual checks that two public keys are equal
func IsPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}

func PubkeyToHex(key *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(key))
}

func HexToPubkey(pk string) (*ecdsa.PublicKey, error) {
	bytes, err := types.DecodeHex(pk)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(bytes)
}
