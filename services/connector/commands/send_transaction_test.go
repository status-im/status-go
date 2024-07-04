package commands

import (
	"encoding/json"
	"testing"

	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/transactions"
)

func TestFailToSendTransactionWithoutDAppPermission(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{
		Db: db,
	}

	sendArgs := transactions.SendTxArgs{
		From:  types.Address{0x01},
		To:    &types.Address{0x02},
		Value: &hexutil.Big{},
		Data:  types.HexBytes("0x0"),
	}
	sendArgsJSON, err := json.Marshal(sendArgs)
	assert.NoError(t, err)

	params := make([]interface{}, 1)
	params[0] = sendArgsJSON

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_sendTransaction",
		Params:      params,
		Origin:      "http://testDAppURL",
		DAppName:    "testDAppName",
		DAppIconUrl: "http://testDAppIconUrl",
	}

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
}

func TestFailToSendTransactionWithWrongFromAddress(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{
		Db: db,
	}

	sendArgs := transactions.SendTxArgs{
		From:  types.Address{0x01},
		To:    &types.Address{0x02},
		Value: &hexutil.Big{},
		Data:  types.HexBytes("0x0"),
	}
	sendArgsJSON, err := json.Marshal(sendArgs)
	assert.NoError(t, err)

	params := make([]interface{}, 1)
	params[0] = string(sendArgsJSON)

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_sendTransaction",
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
		ChainID:       0x1,
	}

	err = persistence.UpsertDApp(db, &dApp)
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrParamsFromAddressIsNotShared, err)
}
