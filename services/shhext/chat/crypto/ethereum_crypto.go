package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto/ecies"

	"github.com/ethereum/go-ethereum/crypto"
	dr "github.com/status-im/doubleratchet"
	"golang.org/x/crypto/hkdf"
)

// EthereumCrypto is an implementation of Crypto with cryptographic primitives recommended
// by the Double Ratchet Algorithm specification. However, some details are different,
// see function comments for details.
type EthereumCrypto struct{}

// GenerateDH; See the Crypto interface.
func (c EthereumCrypto) GenerateDH() (dr.DHPair, error) {
	keys, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	var publicKey [32]byte
	copy(publicKey[:], crypto.CompressPubkey(&keys.PublicKey)[:32])

	var privateKey [32]byte
	copy(privateKey[:], crypto.FromECDSA(keys))

	return DHPair{
		PrvKey: privateKey,
		PubKey: publicKey,
	}, nil

}

// DH; See the Crypto interface.
func (c EthereumCrypto) DH(dhPair dr.DHPair, dhPub dr.Key) dr.Key {
	tmpKey := dhPair.PrivateKey()
	privateKey, err := crypto.ToECDSA(tmpKey[:])
	eciesPrivate := ecies.ImportECDSA(privateKey)
	var a [32]byte
	if err != nil {
		return a
	}

	publicKey, err := crypto.DecompressPubkey(dhPub[:])
	if err != nil {
		return a
	}
	eciesPublic := ecies.ImportECDSAPublic(publicKey)

	key, err := eciesPrivate.GenerateShared(
		eciesPublic,
		16,
		16,
	)

	if err != nil {
		return a
	}

	copy(a[:], key)
	return a

}

// KdfRK; See the Crypto interface.
func (c EthereumCrypto) KdfRK(rk, dhOut dr.Key) (rootKey, chainKey, headerKey dr.Key) {
	var (
		// We can use a non-secret constant as the last argument
		r   = hkdf.New(sha256.New, dhOut[:], rk[:], []byte("rsZUpEuXUqqwXBvSy3EcievAh4cMj6QL"))
		buf = make([]byte, 96)
	)

	// The only error here is an entropy limit which won't be reached for such a short buffer.
	_, _ = io.ReadFull(r, buf)

	copy(rootKey[:], buf[:32])
	copy(chainKey[:], buf[32:64])
	copy(headerKey[:], buf[64:96])
	return
}

// KdfCK; See the Crypto interface.
func (c EthereumCrypto) KdfCK(ck dr.Key) (chainKey dr.Key, msgKey dr.Key) {
	const (
		ckInput = 15
		mkInput = 16
	)

	h := hmac.New(sha256.New, ck[:])

	_, _ = h.Write([]byte{ckInput})
	copy(chainKey[:], h.Sum(nil))
	h.Reset()

	_, _ = h.Write([]byte{mkInput})
	copy(msgKey[:], h.Sum(nil))

	return chainKey, msgKey
}

// Encrypt uses a slightly different approach than in the algorithm specification:
// it uses AES-256-CTR instead of AES-256-CBC for security, ciphertext length and implementation
// complexity considerations.
func (c EthereumCrypto) Encrypt(mk dr.Key, plaintext, ad []byte) []byte {
	encKey, authKey, iv := c.deriveEncKeys(mk)

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	copy(ciphertext, iv[:])

	var (
		block, _ = aes.NewCipher(encKey[:]) // No error will occur here as encKey is guaranteed to be 32 bytes.
		stream   = cipher.NewCTR(block, iv[:])
	)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return append(ciphertext, c.computeSignature(authKey[:], ciphertext, ad)...)
}

// Decrypt; See the Crypto interface.
func (c EthereumCrypto) Decrypt(mk dr.Key, authCiphertext, ad []byte) ([]byte, error) {
	var (
		l          = len(authCiphertext)
		ciphertext = authCiphertext[:l-sha256.Size]
		signature  = authCiphertext[l-sha256.Size:]
	)

	// Check the signature.
	encKey, authKey, _ := c.deriveEncKeys(mk)

	if s := c.computeSignature(authKey[:], ciphertext, ad); !bytes.Equal(s, signature) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decrypt.
	var (
		block, _  = aes.NewCipher(encKey[:]) // No error will occur here as encKey is guaranteed to be 32 bytes.
		stream    = cipher.NewCTR(block, ciphertext[:aes.BlockSize])
		plaintext = make([]byte, len(ciphertext[aes.BlockSize:]))
	)
	stream.XORKeyStream(plaintext, ciphertext[aes.BlockSize:])

	return plaintext, nil
}

// deriveEncKeys derive keys for message encryption and decryption. Returns (encKey, authKey, iv, err).
func (c EthereumCrypto) deriveEncKeys(mk dr.Key) (encKey dr.Key, authKey dr.Key, iv [16]byte) {
	// First, derive encryption and authentication key out of mk.
	salt := make([]byte, 32)
	var (
		r   = hkdf.New(sha256.New, mk[:], salt, []byte("pcwSByyx2CRdryCffXJwy7xgVZWtW5Sh"))
		buf = make([]byte, 80)
	)

	// The only error here is an entropy limit which won't be reached for such a short buffer.
	_, _ = io.ReadFull(r, buf)

	copy(encKey[:], buf[0:32])
	copy(authKey[:], buf[32:64])
	copy(iv[:], buf[64:80])
	return
}

func (c EthereumCrypto) computeSignature(authKey, ciphertext, associatedData []byte) []byte {
	h := hmac.New(sha256.New, authKey)
	_, _ = h.Write(associatedData)
	_, _ = h.Write(ciphertext)
	return h.Sum(nil)
}

type DHPair struct {
	PrvKey dr.Key
	PubKey dr.Key
}

func (p DHPair) PrivateKey() dr.Key {
	return p.PrvKey
}

func (p DHPair) PublicKey() dr.Key {
	return p.PubKey
}
