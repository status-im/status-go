package walletconnect

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewAuth(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)
	require.NotNil(t, auth)
	require.NotEmpty(t, auth.ClientID())
	require.True(t, strings.HasPrefix(auth.ClientID(), "did:key:z"))
}

func TestNewAuth_UniqueKeys(t *testing.T) {
	auth1, err := NewAuth()
	require.NoError(t, err)

	auth2, err := NewAuth()
	require.NoError(t, err)

	require.NotEqual(t, auth1.ClientID(), auth2.ClientID())
}

func TestAuth_ClientID(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	clientID := auth.ClientID()
	require.NotEmpty(t, clientID)
	require.True(t, strings.HasPrefix(clientID, "did:key:z"))
	require.Greater(t, len(clientID), 50)
}

func TestAuth_GenerateJWT(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	aud := "wss://relay.walletconnect.com"
	jwt, err := auth.GenerateJWT(aud)
	require.NoError(t, err)
	require.NotEmpty(t, jwt)

	parts := strings.Split(jwt, ".")
	require.Len(t, parts, 3, "JWT should have header.payload.signature")
}

func TestAuth_GenerateJWT_ValidHeader(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	jwt, err := auth.GenerateJWT("wss://test.com")
	require.NoError(t, err)

	parts := strings.Split(jwt, ".")
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)

	var header map[string]string
	err = json.Unmarshal(headerBytes, &header)
	require.NoError(t, err)
	require.Equal(t, "EdDSA", header["alg"])
	require.Equal(t, "JWT", header["typ"])
}

func TestAuth_GenerateJWT_ValidPayload(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	aud := "wss://relay.example.com"
	before := time.Now().Unix()
	jwt, err := auth.GenerateJWT(aud)
	require.NoError(t, err)
	after := time.Now().Unix()

	parts := strings.Split(jwt, ".")
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var payload map[string]interface{}
	err = json.Unmarshal(payloadBytes, &payload)
	require.NoError(t, err)

	require.Equal(t, auth.ClientID(), payload["iss"])
	require.Equal(t, aud, payload["aud"])
	require.Equal(t, "client_auth", payload["act"])
	require.NotEmpty(t, payload["sub"])

	iat := int64(payload["iat"].(float64))
	exp := int64(payload["exp"].(float64))
	require.GreaterOrEqual(t, iat, before)
	require.LessOrEqual(t, iat, after)
	require.Equal(t, iat+86400, exp)
}

func TestAuth_GenerateJWT_ValidSignature(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	jwt, err := auth.GenerateJWT("wss://test.com")
	require.NoError(t, err)

	parts := strings.Split(jwt, ".")
	message := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	require.Len(t, signature, ed25519.SignatureSize)

	valid := ed25519.Verify(auth.publicKey, []byte(message), signature)
	require.True(t, valid, "JWT signature should be valid")
}

func TestAuth_GenerateJWT_UniqueNonce(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	jwt1, err := auth.GenerateJWT("wss://test.com")
	require.NoError(t, err)

	jwt2, err := auth.GenerateJWT("wss://test.com")
	require.NoError(t, err)

	require.NotEqual(t, jwt1, jwt2, "Each JWT should have unique nonce")
}

func TestAuth_GenerateJWT_DifferentAud(t *testing.T) {
	auth, err := NewAuth()
	require.NoError(t, err)

	jwt1, err := auth.GenerateJWT("wss://relay1.com")
	require.NoError(t, err)

	jwt2, err := auth.GenerateJWT("wss://relay2.com")
	require.NoError(t, err)

	require.NotEqual(t, jwt1, jwt2)

	parts1 := strings.Split(jwt1, ".")
	parts2 := strings.Split(jwt2, ".")
	payload1, err := base64.RawURLEncoding.DecodeString(parts1[1])
	require.NoError(t, err)
	payload2, err := base64.RawURLEncoding.DecodeString(parts2[1])
	require.NoError(t, err)

	var p1, p2 map[string]interface{}
	err = json.Unmarshal(payload1, &p1)
	require.NoError(t, err)
	err = json.Unmarshal(payload2, &p2)
	require.NoError(t, err)

	require.Equal(t, "wss://relay1.com", p1["aud"])
	require.Equal(t, "wss://relay2.com", p2["aud"])
}

func TestEncodeDidKey(t *testing.T) {
	pub := make([]byte, ed25519.PublicKeySize)
	for i := range pub {
		pub[i] = byte(i)
	}

	didKey, err := encodeDidKey(pub)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(didKey, "did:key:z"))
	require.Greater(t, len(didKey), 50)
}

func TestEncodeDidKey_Deterministic(t *testing.T) {
	pub := make([]byte, ed25519.PublicKeySize)

	didKey1, err := encodeDidKey(pub)
	require.NoError(t, err)

	didKey2, err := encodeDidKey(pub)
	require.NoError(t, err)

	require.Equal(t, didKey1, didKey2)
}

func TestEncodeDidKey_DifferentKeys(t *testing.T) {
	pub1 := make([]byte, ed25519.PublicKeySize)
	pub2 := make([]byte, ed25519.PublicKeySize)
	pub2[0] = 1

	didKey1, err := encodeDidKey(pub1)
	require.NoError(t, err)

	didKey2, err := encodeDidKey(pub2)
	require.NoError(t, err)

	require.NotEqual(t, didKey1, didKey2)
}
