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

func prepareSendTransactionRequest(dApp signal.ConnectorDApp, from types.Address) (RPCRequest, error) {
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

	var sendArgsMap map[string]interface{}
	err = json.Unmarshal(sendArgsJSON, &sendArgsMap)
	if err != nil {
		return RPCRequest{}, err
	}

	params := []interface{}{sendArgsMap}

	return ConstructRPCRequest("eth_sendTransaction", params, &dApp)
}

func TestFailToSendTransactionWithoutPermittedDApp(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	// Don't save dApp in the database
	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x1})
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
}

func TestFailToSendTransactionWithWrongAddress(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	cmd := &SendTransactionCommand{Db: db}

	err := PersistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x02})
	assert.NoError(t, err)

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrParamsFromAddressIsNotShared, err)
}

func TestSendTransactionWithSignalTimout(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	clientHandler := NewClientSideHandler()

	cmd := &SendTransactionCommand{
		Db:            db,
		ClientHandler: clientHandler,
	}

	err := PersistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x01})
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, err = cmd.Execute(request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestSendTransactionWithSignalSucceed(t *testing.T) {
	db, close := SetupTestDB(t)
	defer close()

	fakedTransactionHash := types.Hash{0x051}

	clientHandler := NewClientSideHandler()

	cmd := &SendTransactionCommand{
		Db:            db,
		ClientHandler: clientHandler,
	}

	err := PersistDAppData(db, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x01})
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = clientHandler.SendTransactionAccepted(SendTransactionAcceptedArgs{
				Hash: fakedTransactionHash,
			})
			assert.NoError(t, err)
		}
	}))

	response, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, response, fakedTransactionHash.String())
}
