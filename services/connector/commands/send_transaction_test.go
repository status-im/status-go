package commands

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

func testDAppData(db *sql.DB, sharedAccount types.Address) (DAppData, error) {
	dAppData := DAppData{
		Origin:  "http://testDAppURL",
		Name:    "testDAppName",
		IconUrl: "http://testDAppIconUrl",
	}

	dApp := persistence.DApp{
		URL:           dAppData.Origin,
		Name:          dAppData.Name,
		IconURL:       dAppData.IconUrl,
		SharedAccount: sharedAccount,
		ChainID:       0x1,
	}

	return dAppData, persistence.UpsertDApp(db, &dApp)
}

func prepareSendRequest(dApp DAppData, from types.Address) (RPCRequest, error) {
	sendArgs := transactions.SendTxArgs{
		From:  from,
		To:    &types.Address{0x02},
		Value: &hexutil.Big{},
		Data:  types.HexBytes("0x0"),
	}
	sendArgsJSON, err := json.Marshal(sendArgs)
	if err != nil {
		return RPCRequest{}, err
	}

	params := make([]interface{}, 1)
	params[0] = string(sendArgsJSON)

	request := RPCRequest{
		JSONRPC:     "2.0",
		ID:          1,
		Method:      "eth_sendTransaction",
		Params:      params,
		Origin:      dApp.Origin,
		DAppName:    dApp.Name,
		DAppIconUrl: dApp.IconUrl,
	}

	return request, nil
}

func TestFailToSendTransactionWithoutPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	// Don't save dApp in the database
	dAppData := DAppData{
		Origin:  "http://testDAppURL",
		Name:    "testDAppName",
		IconUrl: "http://testDAppIconUrl",
	}

	request, err := prepareSendRequest(dAppData, types.Address{0x1})
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
}

func TestFailToSendTransactionWithWrongAddress(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	dAppData, err := testDAppData(db, types.Address{0x01})
	assert.NoError(t, err)

	request, err := prepareSendRequest(dAppData, types.Address{0x02})
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrParamsFromAddressIsNotShared, err)
}

func TestSendTransactionWithSignalTimout(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	clientHandler := NewClientSideHandler(&RPCClientMock{})

	cmd := &SendTransactionCommand{
		Db:            db,
		ClientHandler: clientHandler,
	}

	dAppData, err := testDAppData(db, types.Address{0x01})
	assert.NoError(t, err)

	request, err := prepareSendRequest(dAppData, types.Address{0x01})
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = cmd.Execute(request)
	assert.Equal(t, err, ErrWalletResponseTimeout)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestSendTransactionWithSignalSucceed(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	fakedTransactionHash := types.Hash{0x051}

	clientHandler := NewClientSideHandler(&RPCClientMock{})

	cmd := &SendTransactionCommand{
		Db:            db,
		ClientHandler: clientHandler,
	}

	dAppData, err := testDAppData(db, types.Address{0x01})
	assert.NoError(t, err)

	request, err := prepareSendRequest(dAppData, types.Address{0x01})
	assert.NoError(t, err)

	go func() {
		signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
			var evt EventType
			err := json.Unmarshal(s, &evt)
			assert.NoError(t, err)

			switch evt.Type {
			case signal.EventConnectorSendTransaction:
				var ev signal.ConnectorSendTransactionSignal
				err := json.Unmarshal(evt.Event, &ev)
				assert.NoError(t, err)

				err = clientHandler.ConnectorSendTransactionFinished(ConnectorSendTransactionFinishedArgs{
					Hash:  fakedTransactionHash,
					Error: nil,
				})
				assert.NoError(t, err)
			}
		}))
	}()

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, response, fakedTransactionHash.String())
}
