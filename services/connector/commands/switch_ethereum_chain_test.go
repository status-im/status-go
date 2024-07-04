package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestFailToSwitchEthereumChainWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "wallet_switchEthereumChain",
		Params:  []interface{}{},
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrRequestMissingDAppData)
	assert.Empty(t, result)
}

func TestFailToSwitchEthereumChainWithNoChainId(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SwitchEthereumChainCommand{Db: db}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "wallet_switchEthereumChain",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	_, err := cmd.Execute(request)
	assert.Equal(t, err, ErrEmptyRPCParams)
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

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "wallet_switchEthereumChain",
		Params:      params,
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	_, err := cmd.Execute(request)
	assert.Equal(t, err, ErrUnsupportedNetwork)
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

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "wallet_switchEthereumChain",
		Params:      params,
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	dApp := persistence.DApp{
		URL:           request.Origin,
		Name:          request.DAppName,
		IconURL:       request.DAppIconUrl,
		SharedAccount: types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"),
		ChainID:       walletCommon.EthereumGoerli,
	}

	err := persistence.UpsertDApp(db, &dApp)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.NotEqual(t, walletCommon.ChainID(dApp.ChainID).String(), response)
	assert.Equal(t, walletCommon.ChainID(walletCommon.EthereumMainnet).String(), response)
}
