package connector

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	"github.com/status-im/status-go/rpc/network"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type testState struct {
	ctx       context.Context
	db        *sql.DB
	walletDb  *sql.DB
	mockCtrl  *gomock.Controller
	rpcClient *mock_rpcclient.MockClientInterface
	service   *Service
	api       *API
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

func setupTests(t *testing.T) (state testState, close func()) {
	state.ctx = context.Background()

	var (
		closeDb       func()
		closeWalletDb func()
	)
	state.db, closeDb = createDB(t)
	state.walletDb, closeWalletDb = createWalletDB(t)

	config := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
	networks := json.RawMessage("{}")
	settingsObj := settings.Settings{
		Networks: &networks,
	}

	settDb, err := settings.MakeNewDB(state.db)
	require.NoError(t, err)
	err = settDb.CreateSettings(settingsObj, config)
	require.NoError(t, err)

	state.mockCtrl = gomock.NewController(t)
	state.rpcClient = mock_rpcclient.NewMockClientInterface(state.mockCtrl)

	networkManager := network.NewManager(state.db)
	require.NotNil(t, networkManager)

	err = networkManager.Init([]params.Network{
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

	state.rpcClient.EXPECT().GetNetworkManager().AnyTimes().Return(networkManager)

	state.service = NewService(state.walletDb, state.rpcClient, state.rpcClient.GetNetworkManager())

	state.api = NewAPI(state.service)

	return state, func() {
		closeDb()
		closeWalletDb()
	}
}
