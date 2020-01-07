package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

func TestMessageID(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyBytes := crypto.FromECDSAPub(&key.PublicKey)

	data := []byte("test")
	expectedID := types.HexBytes(crypto.Keccak256(append(keyBytes, data...)))
	require.Equal(t, expectedID, MessageID(&key.PublicKey, data))
}
