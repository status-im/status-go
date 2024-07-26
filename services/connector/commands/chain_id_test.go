package commands

import (
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/connector/chainutils"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func setupNetworks(db *sql.DB) *ChainIDCommand {
	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
		},
		{
			ChainID: walletCommon.EthereumGoerli,
		},
	})
	cmd := &ChainIDCommand{
		Db:             db,
		NetworkManager: &nm,
	}

	return cmd
}

func TestFailToGetChainIdWithMissingDAppFields(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := setupNetworks(db)

	// Missing DApp fields
	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, nil)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestGetDefaultChainIdForUnpermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := setupNetworks(db)

	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	result, err := cmd.Execute(request)
	assert.NoError(t, err)
	chainId, err := chainutils.GetHexChainID(strconv.FormatUint(walletCommon.EthereumMainnet, 16))
	assert.NoError(t, err)
	assert.Equal(t, result, chainId)
}

func TestGetChainIdForPermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := setupNetworks(db)

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")
	chainID := uint64(0x123)

	err := PersistDAppData(db, testDAppData, sharedAccount, chainID)
	assert.NoError(t, err)

	request, err := ConstructRPCRequest("eth_chainId", []interface{}{}, &testDAppData)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	chainId := fmt.Sprintf(`0x%s`, strconv.FormatUint(chainID, 16))
	assert.Equal(t, chainId, response)
}
