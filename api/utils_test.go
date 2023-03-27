package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/t/utils"
)

func TestHashMessage(t *testing.T) {
	utils.Init()

	backend, stop1, stop2, err := setupGethStatusBackend()
	defer func() {
		err := stop1()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	defer func() {
		err := stop2()
		if err != nil {
			require.NoError(t, backend.StopNode())
		}
	}()
	require.NoError(t, err)

	config, err := utils.MakeTestNodeConfig(params.StatusChainNetworkID)
	require.NoError(t, err)
	require.NoError(t, backend.AccountManager().InitKeystore(config.KeyStoreDir))
	err = backend.StartNode(config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	scenarios := []struct {
		message        string
		expectedHash   string
		recoverMessage string
	}{
		{
			message:        "XYZ",
			expectedHash:   "634349abf2de883d23e8b46972896c7652a06670c990410d3436d9b44db09e6b",
			recoverMessage: fmt.Sprintf("0x%x", "XYZ"),
		},
		{
			message:        "0xXYZ",
			expectedHash:   "f9c57a8998c71a2c8d74d70abe6561838f0d6cb6d82bc85bd70afcc82368055c",
			recoverMessage: fmt.Sprintf("0x%x", "0xXYZ"),
		},
		{
			message:        "1122",
			expectedHash:   "3f07e02a153f02bdf97d77161746257626e9c39e4c3cf59896365fd1e6a9c7c3",
			recoverMessage: fmt.Sprintf("0x%x", "1122"),
		},
		{
			message:        "0x1122",
			expectedHash:   "86d79d0957efa9b7d91f1116e70d0ee934cb9cdeccefa07756aed2bee119a2f3",
			recoverMessage: "0x1122",
		},
	}

	for _, s := range scenarios {
		t.Run(s.message, func(t *testing.T) {
			hash, err := HashMessage(s.message)
			require.Nil(t, err)
			require.Equal(t, s.expectedHash, fmt.Sprintf("%x", hash))

			// simulate signature from external signer like a keycard
			sig, err := crypto.Sign(hash, key)
			require.NoError(t, err)
			sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

			// check that the message was wrapped correctly before hashing it
			recParams := personal.RecoverParams{
				Message:   s.recoverMessage,
				Signature: fmt.Sprintf("0x%x", sig),
			}

			recoveredAddr, err := backend.Recover(recParams)
			require.NoError(t, err)
			assert.Equal(t, addr, recoveredAddr)
		})
	}
}
