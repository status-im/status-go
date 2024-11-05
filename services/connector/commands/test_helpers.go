package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	"github.com/status-im/status-go/rpc/network"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

var testDAppData = signal.ConnectorDApp{
	URL:     "http://testDAppURL",
	Name:    "testDAppName",
	IconURL: "http://testDAppIconUrl",
}

type EventType struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

func createDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "browser-connect-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func createWalletDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

type testState struct {
	ctx       context.Context
	db        *sql.DB
	walletDb  *sql.DB
	cmd       RPCCommand
	handler   *ClientSideHandler
	mockCtrl  *gomock.Controller
	rpcClient *mock_rpcclient.MockClientInterface
}

func setupCommand(t *testing.T, method string) (state testState, close func()) {
	state.ctx = context.Background()

	var (
		closeDb       func()
		closeWalletDb func()
	)
	state.db, closeDb = createDB(t)
	state.walletDb, closeWalletDb = createWalletDB(t)

	networkManager := network.NewManager(state.db)
	require.NotNil(t, networkManager)

	err := networkManager.Init([]params.Network{
		{
			ChainID: walletCommon.EthereumMainnet,
			Layer:   1,
		},
		{
			ChainID: walletCommon.OptimismMainnet,
			Layer:   1,
		},
	})
	require.NoError(t, err)

	state.handler = NewClientSideHandler(state.db)

	state.mockCtrl = gomock.NewController(t)
	state.rpcClient = mock_rpcclient.NewMockClientInterface(state.mockCtrl)

	switch method {
	case Method_EthAccounts:
		state.cmd = &AccountsCommand{
			Db: state.walletDb,
		}
	case Method_EthRequestAccounts:
		state.cmd = &RequestAccountsCommand{
			ClientHandler: state.handler,
			Db:            state.walletDb,
		}
	case Method_EthChainId:
		state.cmd = &ChainIDCommand{
			Db:             state.walletDb,
			NetworkManager: networkManager,
		}
	case Method_PersonalSign:
		state.cmd = &PersonalSignCommand{
			Db:            state.walletDb,
			ClientHandler: state.handler,
		}
	case Method_EthSendTransaction:
		state.cmd = &SendTransactionCommand{
			Db:            state.walletDb,
			ClientHandler: state.handler,
			RpcClient:     state.rpcClient,
		}
	case Method_RequestPermissions:
		state.cmd = &RequestPermissionsCommand{}
	case Method_RevokePermissions:
		state.cmd = &RevokePermissionsCommand{
			Db: state.walletDb,
		}
	case Method_SwitchEthereumChain:
		state.cmd = &SwitchEthereumChainCommand{
			Db:             state.walletDb,
			NetworkManager: networkManager,
		}
	}

	return state, func() {
		closeDb()
		closeWalletDb()
	}
}

func PersistDAppData(db *sql.DB, dApp signal.ConnectorDApp, sharedAccount types.Address, chainID uint64) error {
	dAppDb := persistence.DApp{
		URL:           dApp.URL,
		Name:          dApp.Name,
		IconURL:       dApp.IconURL,
		SharedAccount: sharedAccount,
		ChainID:       chainID,
	}

	return persistence.UpsertDApp(db, &dAppDb)
}

func ConstructRPCRequest(method string, params []interface{}, dApp *signal.ConnectorDApp) (RPCRequest, error) {
	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	if dApp != nil {
		request.URL = dApp.URL
		request.Name = dApp.Name
		request.IconURL = dApp.IconURL
	}

	return request, nil
}
