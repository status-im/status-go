package wallet

import (
	"context"
	"encoding/json"
	"testing"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdb"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	network_mock "github.com/status-im/status-go/pkg/services/networks/mock"
	"github.com/status-im/status-go/pkg/services/wallet/token"
)

func TestAPI_GetTokenBySymbolOnChainOverRPC(t *testing.T) {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	defer appDB.Close()
	require.NoError(t, protocolsqlite.Migrate(appDB))
	walletDB, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	defer walletDB.Close()
	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)
	settingsDB, err := settings.MakeNewDB(appDB)
	require.NoError(t, err)
	networks := json.RawMessage(`{}`)
	require.NoError(t, settingsDB.CreateSettings(settings.Settings{Networks: &networks, ThirdpartyServicesEnabled: false}, params.NodeConfig{}))

	mockCtrl := gomock.NewController(t)
	networkManager := network_mock.NewMockManagerInterface(mockCtrl)
	networkManager.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: 1}, {ChainID: 10}}, nil).AnyTimes()
	networkManager.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	tokenManager, err := token.NewTokenManager(walletDB, nil, nil, networkManager, appDB, nil, nil, pubsub.NewPublisher(), accountsDB, 0, 0)
	require.NoError(t, err)
	require.NoError(t, tokenManager.Start(context.Background()))
	defer tokenManager.Stop()

	server := gethrpc.NewServer()
	defer server.Stop()
	require.NoError(t, server.RegisterName("wallet", NewAPI(&Service{tokenManager: tokenManager})))
	client := gethrpc.DialInProc(server)
	defer client.Close()

	var byChain []json.RawMessage
	require.NoError(t, client.Call(&byChain, "wallet_getTokensByChain", 1))
	var usdc json.RawMessage
	for _, raw := range byChain {
		var fields struct{ Symbol string }
		require.NoError(t, json.Unmarshal(raw, &fields))
		if fields.Symbol == "USDC" {
			usdc = raw
			break
		}
	}
	require.NotNil(t, usdc)

	var found json.RawMessage
	require.NoError(t, client.Call(&found, "wallet_getTokenBySymbolOnChain", 1, "usdc"))
	require.JSONEq(t, string(usdc), string(found), "same shape as wallet_getTokensByChain")

	found = nil
	require.NoError(t, client.Call(&found, "wallet_getTokenBySymbolOnChain", 1, "no-such-token"))
	require.Equal(t, "null", string(found))
}
