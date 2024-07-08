package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestFailToGetChainIdWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{Db: db}

	// Missing DApp fields
	request := constructRPCRequest("eth_chainId", []interface{}{}, nil)

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrRequestMissingDAppData)
	assert.Empty(t, result)
}

func TestFailToGetChainIdForUnpermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{Db: db}

	request := constructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrDAppIsNotPermittedByUser)
	assert.Empty(t, result)
}

func TestGetChainIdForPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{Db: db}

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")
	chainID := uint64(0x123)

	err := persistDAppData(db, testDAppData, sharedAccount, chainID)
	assert.NoError(t, err)

	request := constructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, walletCommon.ChainID(chainID).String(), response)
}
