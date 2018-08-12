package doubleratchet

import "encoding/hex"

// Crypto is a cryptography supplement for the library.
type Crypto interface {
	// GenerateDH creates a new Diffie-Hellman key pair.
	GenerateDH() (DHPair, error)

	// DH returns the output from the Diffie-Hellman calculation between
	// the private key from the DH key pair dhPair and the DH public key dbPub.
	DH(dhPair DHPair, dhPub Key) Key

	// Encrypt returns an AEAD encryption of plaintext with message key mk. The associated_data
	// is authenticated but is not included in the ciphertext. The AEAD nonce may be set to a constant.
	Encrypt(mk Key, plaintext, ad []byte) (authCiphertext []byte)

	// Decrypt returns the AEAD decryption of ciphertext with message key mk.
	Decrypt(mk Key, ciphertext, ad []byte) (plaintext []byte, err error)

	KDFer
}

// DHPair is a general interface for DH pairs representation.
type DHPair interface {
	PrivateKey() Key
	PublicKey() Key
}

// Key is any 32-byte key. It's created for the possibility of pretty hex output.
type Key [32]byte

// Stringer interface compliance.
func (k Key) String() string {
	return hex.EncodeToString(k[:])
}
