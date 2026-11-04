package noise

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
	n "github.com/waku-org/go-noise"
)

func generateRandomBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func TestQR(t *testing.T) {
	staticKey, _ := n.DH25519.GenerateKeypair()
	ephemeralKey, _ := n.DH25519.GenerateKeypair()
	r := generateRandomBytes(t, 32)
	committedStaticKey := n.CommitPublicKey(sha256.New, staticKey.Public, r)

	// Content topic information
	applicationName := "waku-noise-sessions"
	applicationVersion := "0.1"
	shardID := "10"

	qr := NewQR(applicationName, applicationVersion, shardID, ephemeralKey.Public, committedStaticKey)
	readQR, err := StringToQR(qr.String())
	require.NoError(t, err)

	// We check if QR serialization/deserialization works
	require.Equal(t, applicationName, readQR.applicationName)
	require.Equal(t, applicationVersion, readQR.applicationVersion)
	require.Equal(t, shardID, readQR.shardID)
	require.True(t, bytes.Equal(ephemeralKey.Public, readQR.ephemeralPublicKey))
	require.True(t, bytes.Equal(committedStaticKey[:], readQR.committedStaticKey[:]))
}
