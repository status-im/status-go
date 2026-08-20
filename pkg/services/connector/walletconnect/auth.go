package walletconnect

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/multiformats/go-multibase"
)

const (
	// Ed25519 multicodec prefix
	ed25519MulticodecPrefix = "\xed\x01"

	// JWT constants
	jwtAlg       = "EdDSA"
	jwtTyp       = "JWT"
	jwtDelimiter = "."

	// JWT claim constants
	jwtActClientAuth = "client_auth"
	jwtTTL           = 86400 // 24 hours in seconds
)

// Auth handles Ed25519 keypair management and DID-JWT signing for relay authentication.
type Auth struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	clientID   string // did:key representation
}

// NewAuth creates a new Auth instance with a fresh Ed25519 keypair.
func NewAuth() (*Auth, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 keypair: %w", err)
	}

	clientID, err := encodeDidKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("encode did:key: %w", err)
	}

	return &Auth{
		privateKey: privateKey,
		publicKey:  publicKey,
		clientID:   clientID,
	}, nil
}

// ClientID returns the did:key representation of the public key.
func (a *Auth) ClientID() string {
	return a.clientID
}

// GenerateJWT creates a signed JWT for relay authentication.
// aud should be the relay server URL (e.g. "wss://relay.walletconnect.com")
func (a *Auth) GenerateJWT(aud string) (string, error) {
	now := time.Now().Unix()

	// Generate random nonce as subject
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sub := fmt.Sprintf("%x", nonce)

	// Build JWT header
	header := map[string]string{
		"alg": jwtAlg,
		"typ": jwtTyp,
	}

	// Build JWT payload
	payload := map[string]interface{}{
		"iat": now,
		"exp": now + jwtTTL,
		"iss": a.clientID,
		"aud": aud,
		"sub": sub,
		"act": jwtActClientAuth,
	}

	// Encode header and payload
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Sign the message
	message := headerB64 + jwtDelimiter + payloadB64
	signature := ed25519.Sign(a.privateKey, []byte(message))
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	jwt := message + jwtDelimiter + signatureB64
	return jwt, nil
}

// encodeDidKey encodes an Ed25519 public key as a did:key identifier.
// Format: did:key:z + base58btc(0xed 0x01 + publicKey)
func encodeDidKey(publicKey ed25519.PublicKey) (string, error) {
	// Prepend Ed25519 multicodec prefix (0xed 0x01)
	prefixedKey := append([]byte(ed25519MulticodecPrefix), publicKey...)

	// Encode as base58btc with 'z' prefix
	encoded, err := multibase.Encode(multibase.Base58BTC, prefixedKey)
	if err != nil {
		return "", fmt.Errorf("multibase encode: %w", err)
	}

	// Build did:key identifier
	return "did:key:" + encoded, nil
}
