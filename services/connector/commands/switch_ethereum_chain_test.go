package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestFailToSwitchEthereumChainWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	// Missing DApp fields
	request := constructRPCRequest("wallet_switchEthereumChain", []interface{}{}, nil)

	result, err := cmd.Execute(request)
	assert.Equal(t, ErrRequestMissingDAppData, err)
	assert.Empty(t, result)
}

func TestFailToSwitchEthereumChainWithNoChainId(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	request := constructRPCRequest("wallet_switchEthereumChain", []interface{}{}, &testDAppData)

	_, err := cmd.Execute(request)
	assert.Equal(t, ErrEmptyRPCParams, err)
}

func TestFailToSwitchEthereumChainWithUnsupportedChainId(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
		},
	})

	cmd := &SwitchEthereumChainCommand{
		Db:             db,
		NetworkManager: &nm,
	}

	params := make([]interface{}, 1)
	params[0] = walletCommon.BinanceTestChainID // some unrecoginzed chain id

	request := constructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)

	_, err := cmd.Execute(request)
	assert.Equal(t, ErrUnsupportedNetwork, err)
}

func TestSwitchEthereumChain(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	nm := NetworkManagerMock{}
	nm.SetNetworks([]*params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
		},
		{
			ChainID: walletCommon.EthereumGoerli,
		},
	})

	cmd := &SwitchEthereumChainCommand{
		Db:             db,
		NetworkManager: &nm,
	}

	params := make([]interface{}, 1)
	params[0] = walletCommon.EthereumMainnet

	request := constructRPCRequest("wallet_switchEthereumChain", params, &testDAppData)

	err := persistDAppData(db, testDAppData, types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"), walletCommon.EthereumMainnet)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, walletCommon.ChainID(walletCommon.EthereumMainnet).String(), response)
}
