package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestFailToGetChainIdWithMissingDAppFields(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_chainId",
		Params:  []interface{}{},
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrRequestMissingDAppData)
	assert.Empty(t, result)
}

func TestFailToGetChainIdForUnpermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_chainId",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	result, err := cmd.Execute(request)
	assert.Equal(t, err, ErrDAppIsNotPermittedByUser)
	assert.Empty(t, result)
}

func TestGetChainIdForPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &ChainIDCommand{
		Db: db,
	}

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_chainId",
		Params:      []interface{}{},
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	dApp := persistence.DApp{
		URL:           request.Origin,
		Name:          request.DAppName,
		IconURL:       request.DAppIconUrl,
		SharedAccount: types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"),
		ChainID:       0x1,
	}

	err := persistence.UpsertDApp(db, &dApp)
	assert.NoError(t, err)

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, walletCommon.ChainID(dApp.ChainID).String(), response)
}
