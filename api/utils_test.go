package api

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashMessage(t *testing.T) {
	backend := NewStatusBackend()
	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	originalMessage := "hello world"
	hash := HashMessage(originalMessage)

	// simulate signature from external signer like a keycard
	sig, err := crypto.Sign(hash, key)
	require.NoError(t, err)
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	// check that the message was wrapped correctly before hashing it
	recParams := personal.RecoverParams{
		Message:   fmt.Sprintf("0x%x", originalMessage),
		Signature: fmt.Sprintf("0x%x", sig),
	}
	recoveredAddr, err := backend.Recover(recParams)
	require.NoError(t, err)
	assert.Equal(t, addr, recoveredAddr)
}
