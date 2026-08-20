package connector

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/rpc/network"
	network_testutil "github.com/status-im/status-go/internal/rpc/network/testutil"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	mock_chainutils "github.com/status-im/status-go/pkg/services/connector/chainutils/mock"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

type testState struct {
	ctx             context.Context
	db              *sql.DB
	walletDb        *sql.DB
	mockCtrl        *gomock.Controller
	ethClientGetter *mock_chainutils.MockEthClientGetter
	feeManager      *mock_chainutils.MockFeeManager
	service         *Service
	api             *API
}

func createDB(t *testing.T) *sql.DB {
	db, cleanup, err := testutils.SetupTestSQLDB(appdatabase.DbInitializer{}, "browser-connect-tests-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, cleanup())
	})
	return db
}

func createWalletDB(t *testing.T) (db *sql.DB) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}

func setupTests(t *testing.T) (state testState) {
	state.ctx = WithConnectionType(context.Background(), ConnectionTypeUntrusted)

	state.db = createDB(t)
	state.walletDb = createWalletDB(t)

	config := params.NodeConfig{
		NetworkID: 10,
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

	networkManager := network.NewManager(state.db, nil)
	require.NotNil(t, networkManager)

	initNetworks := []params.Network{
		*network_testutil.CreateNetwork(walletCommon.EthereumMainnet, "Ethereum Mainnet", []params.RpcProvider{
			network_testutil.CreateProvider(walletCommon.EthereumMainnet, "Infura Mainnet", params.EmbeddedEthRpcProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
		*network_testutil.CreateNetwork(walletCommon.OptimismMainnet, "Optimism Mainnet", []params.RpcProvider{
			network_testutil.CreateProvider(walletCommon.OptimismMainnet, "Optimism Mainnet", params.EmbeddedEthRpcProxyProviderType, true, security.NewSensitiveString("https://mainnet.optimism.io")),
		}),
	}
	err = networkManager.InitEmbeddedNetworks(initNetworks)
	require.NoError(t, err)

	state.ethClientGetter = mock_chainutils.NewMockEthClientGetter(state.mockCtrl)
	state.feeManager = mock_chainutils.NewMockFeeManager(state.mockCtrl)

	state.service = NewService(
		zap.NewNop(),
		state.walletDb,
		state.ethClientGetter,
		state.feeManager,
		networkManager,
		&Config{},
	)

	state.api = state.service.api

	return state
}
