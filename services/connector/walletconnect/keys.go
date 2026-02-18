package walletconnect

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	// X25519 keys are 32 bytes
	x25519KeySize = 32
)

// GenerateX25519KeyPair generates a new X25519 key pair for WalletConnect session key agreement.
// Returns privateKey and publicKey as raw bytes (32 bytes each).
func GenerateX25519KeyPair() (privateKey, publicKey []byte, err error) {
	privateKey = make([]byte, x25519KeySize)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, nil, fmt.Errorf("generate private key: %w", err)
	}

	// Clamp the private key for X25519 (required by the curve)
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	publicKey, err = curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("derive public key: %w", err)
	}

	return privateKey, publicKey, nil
}

// DeriveSharedSecret performs ECDH key agreement using X25519.
// ourPrivate: our 32-byte private key
// theirPublic: peer's 32-byte public key (hex-decoded if passed as hex string)
// Returns the raw 32-byte shared secret.
func DeriveSharedSecret(ourPrivate, theirPublic []byte) ([]byte, error) {
	if len(ourPrivate) != x25519KeySize {
		return nil, fmt.Errorf("invalid ourPrivate length: expected %d, got %d", x25519KeySize, len(ourPrivate))
	}
	if len(theirPublic) != x25519KeySize {
		return nil, fmt.Errorf("invalid theirPublic length: expected %d, got %d", x25519KeySize, len(theirPublic))
	}

	sharedSecret, err := curve25519.X25519(ourPrivate, theirPublic)
	if err != nil {
		return nil, fmt.Errorf("x25519 key exchange: %w", err)
	}
	return sharedSecret, nil
}

// DeriveSymmetricKey derives a 32-byte symmetric key from the shared secret using HKDF-SHA256.
// WalletConnect spec: no salt, no info, 32-byte output.
func DeriveSymmetricKey(sharedSecret []byte) ([]byte, error) {
	if len(sharedSecret) == 0 {
		return nil, fmt.Errorf("shared secret cannot be empty")
	}

	r := hkdf.New(sha256.New, sharedSecret, nil, nil)
	symKey := make([]byte, 32)
	n, err := r.Read(symKey)
	if err != nil {
		return nil, fmt.Errorf("hkdf read: %w", err)
	}
	if n != 32 {
		return nil, fmt.Errorf("hkdf produced %d bytes, expected 32", n)
	}
	return symKey, nil
}

// DeriveSessionTopic computes the session topic from the symmetric key.
// Topic = hex(SHA256(symmetricKey)) as per WalletConnect spec.
func DeriveSessionTopic(symmetricKey []byte) string {
	hash := sha256.Sum256(symmetricKey)
	return hex.EncodeToString(hash[:])
}
