package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

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

	return constructRPCRequest("eth_sendTransaction", params, &dApp), nil
}

func TestFailToSendTransactionWithoutPermittedDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	// Don't save dApp in the database
	request, err := prepareSendRequest(testDAppData, types.Address{0x1})
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
}

func TestFailToSendTransactionWithWrongAddress(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	err := persistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendRequest(testDAppData, types.Address{0x02})
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

	err := persistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendRequest(testDAppData, types.Address{0x01})
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

	err := persistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendRequest(testDAppData, types.Address{0x01})
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
