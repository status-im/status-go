package walletconnect

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// Envelope type identifiers
	envelopeType0 = 0x00 // ChaCha20-Poly1305 with shared symmetric key

	// Type 0 envelope structure
	type0IVOffset   = 1
	type0IVLength   = 12
	type0DataOffset = 13
)

// DecryptType0Envelope decrypts a Type 0 envelope (ChaCha20-Poly1305).
// envelopeB64 is the base64-encoded envelope string from the relay.
// symKeyHex is the hex-encoded 32-byte symmetric key from the WC URI.
//
// Type 0 envelope structure:
// - byte 0: type (0x00)
// - bytes 1-12: IV (12 bytes)
// - bytes 13-N: sealbox (ciphertext + 16-byte Poly1305 tag)
func DecryptType0Envelope(symKeyHex string, envelopeB64 string) ([]byte, error) {
	// Decode symmetric key from hex
	symKey, err := hex.DecodeString(symKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode symkey: %w", err)
	}

	if len(symKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid symkey length: expected %d, got %d", chacha20poly1305.KeySize, len(symKey))
	}

	// Base64-decode the envelope
	envelope, err := base64.StdEncoding.DecodeString(envelopeB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode envelope: %w", err)
	}

	// Verify minimum length (type + IV + tag)
	minLength := 1 + type0IVLength + chacha20poly1305.Overhead
	if len(envelope) < minLength {
		return nil, fmt.Errorf("envelope too short: expected at least %d bytes, got %d", minLength, len(envelope))
	}

	// Verify envelope type
	if envelope[0] != envelopeType0 {
		return nil, fmt.Errorf("unsupported envelope type: 0x%02x (expected 0x%02x)", envelope[0], envelopeType0)
	}

	// Extract IV and sealbox
	iv := envelope[type0IVOffset:type0DataOffset]
	sealbox := envelope[type0DataOffset:]

	// Create ChaCha20-Poly1305 AEAD cipher
	aead, err := chacha20poly1305.New(symKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, iv, sealbox, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptType0Envelope encrypts plaintext into a Type 0 envelope (ChaCha20-Poly1305).
// symKeyHex is the hex-encoded 32-byte symmetric key.
// Returns base64-encoded envelope string suitable for relay publishing.
//
// Type 0 envelope structure:
// - byte 0: type (0x00)
// - bytes 1-12: IV (12 bytes, randomly generated)
// - bytes 13-N: sealbox (ciphertext + 16-byte Poly1305 tag)
func EncryptType0Envelope(symKeyHex string, plaintext []byte) (string, error) {
	symKey, err := hex.DecodeString(symKeyHex)
	if err != nil {
		return "", fmt.Errorf("decode symkey: %w", err)
	}

	if len(symKey) != chacha20poly1305.KeySize {
		return "", fmt.Errorf("invalid symkey length: expected %d, got %d", chacha20poly1305.KeySize, len(symKey))
	}

	aead, err := chacha20poly1305.New(symKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	// Generate random 12-byte IV
	iv := make([]byte, type0IVLength)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("generate iv: %w", err)
	}

	// Seal: encrypt and authenticate
	sealbox := aead.Seal(nil, iv, plaintext, nil)

	// Build envelope: type byte + IV + sealbox
	envelope := make([]byte, 0, 1+type0IVLength+len(sealbox))
	envelope = append(envelope, envelopeType0)
	envelope = append(envelope, iv...)
	envelope = append(envelope, sealbox...)

	return base64.StdEncoding.EncodeToString(envelope), nil
}
