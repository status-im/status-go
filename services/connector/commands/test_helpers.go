package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/rpc/network"
	network_testutil "github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/services/connector/chainutils"
	mock_chainutils "github.com/status-im/status-go/services/connector/chainutils/mock"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
)

var testDAppData = signal.ConnectorDApp{
	URL:      "http://testDAppURL",
	Name:     "testDAppName",
	IconURL:  "http://testDAppIconUrl",
	ClientID: "",
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
	ctx             context.Context
	db              *sql.DB
	walletDb        *sql.DB
	cmd             RPCCommand
	handler         *ClientSideHandler
	mockCtrl        *gomock.Controller
	ethClientGetter *mock_chainutils.MockEthClientGetter
	feeManager      *mock_chainutils.MockFeeManager
}

func setupCommand(t *testing.T, method string) (state testState, close func()) {
	state.ctx = context.Background()

	var (
		closeDb       func()
		closeWalletDb func()
	)
	state.db, closeDb = createDB(t)
	state.walletDb, closeWalletDb = createWalletDB(t)

	networkManager := network.NewManager(state.db, nil)
	require.NotNil(t, networkManager)

	initNetworks := []params.Network{
		*network_testutil.CreateNetwork(walletCommon.EthereumMainnet, "Ethereum Mainnet", []params.RpcProvider{
			network_testutil.CreateProvider(walletCommon.EthereumMainnet, "Infura Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
		*network_testutil.CreateNetwork(walletCommon.OptimismMainnet, "Optimism Mainnet", []params.RpcProvider{
			network_testutil.CreateProvider(walletCommon.OptimismMainnet, "Optimism Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.optimism.io")),
		}),
	}
	err := networkManager.InitEmbeddedNetworks(initNetworks)
	require.NoError(t, err)

	state.handler = NewClientSideHandler(state.db)

	state.mockCtrl = gomock.NewController(t)
	state.ethClientGetter = mock_chainutils.NewMockEthClientGetter(state.mockCtrl)
	state.feeManager = mock_chainutils.NewMockFeeManager(state.mockCtrl)

	switch method {
	case Method_EthAccounts:
		state.cmd = NewAccountsCommand(state.walletDb)
	case Method_EthRequestAccounts:
		state.cmd = NewRequestAccountsCommand(state.walletDb, state.handler)
	case Method_EthChainId:
		defaultChainIDGetter := chainutils.NewNetworkManagerAdapter(networkManager)
		state.cmd = NewChainIDCommand(state.walletDb, defaultChainIDGetter)
	case "net_version":
		defaultChainIDGetter := chainutils.NewNetworkManagerAdapter(networkManager)
		state.cmd = NewNetVersionCommand(state.walletDb, defaultChainIDGetter)
	case Method_PersonalSign:
		state.cmd = NewSignCommand(state.walletDb, state.handler)
	case Method_SignTypedDataV4:
		state.cmd = NewSignCommand(state.walletDb, state.handler)
	case Method_EthSendTransaction:
		state.cmd = NewSendTransactionCommand(state.walletDb, state.ethClientGetter, state.feeManager, state.handler)
	case Method_RequestPermissions:
		state.cmd = NewRequestPermissionsCommand(state.walletDb)
	case Method_RevokePermissions:
		state.cmd = NewRevokePermissionsCommand(state.walletDb)
	case Method_SwitchEthereumChain:
		state.cmd = NewSwitchEthereumChainCommand(state.walletDb, networkManager)
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
		ClientID:      dApp.ClientID,
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
		request.ClientID = dApp.ClientID
	}

	return request, nil
}
