package walletconnect

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateX25519KeyPair(t *testing.T) {
	priv, pub, err := GenerateX25519KeyPair()
	require.NoError(t, err)
	require.Len(t, priv, 32)
	require.Len(t, pub, 32)
	require.NotEqual(t, priv, pub)
}

func TestGenerateX25519KeyPair_Clamping(t *testing.T) {
	priv, _, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	require.Equal(t, byte(0), priv[0]&7, "bits 0-2 should be zero")
	require.Equal(t, byte(0), priv[31]&128, "bit 255 should be zero")
	require.Equal(t, byte(64), priv[31]&64, "bit 254 should be one")
}

func TestGenerateX25519KeyPair_Unique(t *testing.T) {
	priv1, pub1, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	priv2, pub2, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	require.NotEqual(t, priv1, priv2)
	require.NotEqual(t, pub1, pub2)
}

func TestDeriveSharedSecret(t *testing.T) {
	priv1, pub1, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	priv2, pub2, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	secret1, err := DeriveSharedSecret(priv1, pub2)
	require.NoError(t, err)
	require.Len(t, secret1, 32)

	secret2, err := DeriveSharedSecret(priv2, pub1)
	require.NoError(t, err)
	require.Len(t, secret2, 32)

	require.Equal(t, secret1, secret2, "ECDH should produce same shared secret")
}

func TestDeriveSharedSecret_InvalidPrivateKeyLength(t *testing.T) {
	_, err := DeriveSharedSecret([]byte{1, 2, 3}, make([]byte, 32))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid ourPrivate length")
}

func TestDeriveSharedSecret_InvalidPublicKeyLength(t *testing.T) {
	_, err := DeriveSharedSecret(make([]byte, 32), []byte{1, 2, 3})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid theirPublic length")
}

func TestDeriveSharedSecret_ZeroKeys(t *testing.T) {
	priv := make([]byte, 32)
	pub := make([]byte, 32)

	_, err := DeriveSharedSecret(priv, pub)
	require.Error(t, err, "should reject low-order point (all-zero public key)")
	require.Contains(t, err.Error(), "low order point")
}

func TestDeriveSymmetricKey(t *testing.T) {
	sharedSecret := []byte("test shared secret of any length")

	symKey, err := DeriveSymmetricKey(sharedSecret)
	require.NoError(t, err)
	require.Len(t, symKey, 32)
}

func TestDeriveSymmetricKey_EmptySecret(t *testing.T) {
	_, err := DeriveSymmetricKey([]byte{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "shared secret cannot be empty")
}

func TestDeriveSymmetricKey_Deterministic(t *testing.T) {
	sharedSecret := []byte("deterministic input")

	symKey1, err := DeriveSymmetricKey(sharedSecret)
	require.NoError(t, err)

	symKey2, err := DeriveSymmetricKey(sharedSecret)
	require.NoError(t, err)

	require.Equal(t, symKey1, symKey2, "HKDF should be deterministic")
}

func TestDeriveSymmetricKey_DifferentInputs(t *testing.T) {
	symKey1, err := DeriveSymmetricKey([]byte("input1"))
	require.NoError(t, err)

	symKey2, err := DeriveSymmetricKey([]byte("input2"))
	require.NoError(t, err)

	require.NotEqual(t, symKey1, symKey2)
}

func TestDeriveSessionTopic(t *testing.T) {
	symKey := []byte("test symmetric key 32 bytes!!")

	topic := DeriveSessionTopic(symKey)
	require.Len(t, topic, 64, "SHA256 hex should be 64 chars")

	_, err := hex.DecodeString(topic)
	require.NoError(t, err, "topic should be valid hex")
}

func TestDeriveSessionTopic_Deterministic(t *testing.T) {
	symKey := []byte("same key")

	topic1 := DeriveSessionTopic(symKey)
	topic2 := DeriveSessionTopic(symKey)

	require.Equal(t, topic1, topic2)
}

func TestDeriveSessionTopic_DifferentKeys(t *testing.T) {
	topic1 := DeriveSessionTopic([]byte("key1"))
	topic2 := DeriveSessionTopic([]byte("key2"))

	require.NotEqual(t, topic1, topic2)
}

func TestDeriveSessionTopic_MatchesSpec(t *testing.T) {
	symKey := []byte("test key")
	expected := sha256.Sum256(symKey)
	expectedHex := hex.EncodeToString(expected[:])

	topic := DeriveSessionTopic(symKey)
	require.Equal(t, expectedHex, topic)
}

func TestKeyAgreementFullFlow(t *testing.T) {
	priv1, pub1, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	priv2, pub2, err := GenerateX25519KeyPair()
	require.NoError(t, err)

	secret1, err := DeriveSharedSecret(priv1, pub2)
	require.NoError(t, err)

	secret2, err := DeriveSharedSecret(priv2, pub1)
	require.NoError(t, err)

	require.Equal(t, secret1, secret2)

	symKey1, err := DeriveSymmetricKey(secret1)
	require.NoError(t, err)

	symKey2, err := DeriveSymmetricKey(secret2)
	require.NoError(t, err)

	require.Equal(t, symKey1, symKey2)

	topic1 := DeriveSessionTopic(symKey1)
	topic2 := DeriveSessionTopic(symKey2)

	require.Equal(t, topic1, topic2)
}
