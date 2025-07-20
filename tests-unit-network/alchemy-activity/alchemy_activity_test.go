package alchemy_activity_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	geth_common "github.com/ethereum/go-ethereum/common"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/api"
	api_common "github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
	"github.com/status-im/status-go/t/helpers"
	t_common "github.com/status-im/status-go/tests-unit-network/common"
	_ "github.com/waku-org/go-zerokit-rln-apple/rln"
)

func setupAlchemyActivityClient(t *testing.T) *alchemy.Client {
	appDB, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "alchemy-activity-tests")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })

	walletSecrets := t_common.GetWalletSecretsConfigFromEnv()
	defaultNetworks := api.BuildDefaultNetworks(walletSecrets)
	networkManager := network.NewManager(appDB, nil, nil, nil)
	err = networkManager.InitEmbeddedNetworks(defaultNetworks)
	require.NoError(t, err)

	config := rpc.ClientConfig{
		Networks: defaultNetworks,
		DB:       appDB,
	}
	rpcClient, err := rpc.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	rpcClient.Start(ctx)
	t.Cleanup(func() { rpcClient.Stop() })

	alchemyEthClientGetter := rpc.NewProviderChainClientGetter(api_common.SmartProxyAlchemy, rpcClient)

	alchemyActivityClient := alchemy.NewClient(alchemyEthClientGetter)

	return alchemyActivityClient
}

func TestFetchHistoryBoth(t *testing.T) {
	alchemyActivityClient := setupAlchemyActivityClient(t)

	// address := geth_common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	address := geth_common.HexToAddress("0xe0798E0070D223beD269267BBA11f4aBFfE27a88")
	// Expecting these transfers:
	// https://etherscan.io/advanced-filter?fadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e&tadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e&age=2024-07-22%7e2024-07-22
	// fromBlock := geth_rpc.BlockNumber(20364031)
	// toBlock := geth_rpc.BlockNumber(20364204)

	parameters := thirdparty.ActivityFetchParameters{
		Address:   address, // vitalik.eth
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Both,
		// FromBlock: &fromBlock,
		// ToBlock:   &toBlock,
	}

	history, err := alchemyActivityClient.FetchActivity(context.Background(), api_common.MainnetChainID, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 11, len(history.Items))
}

func TestFetchHistoryIncoming(t *testing.T) {
	alchemyActivityClient := setupAlchemyActivityClient(t)

	address := geth_common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	// Expecting these transfers:
	// https://etherscan.io/advanced-filter?tadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e&age=2024-07-22%7e2024-07-22
	fromBlock := geth_rpc.BlockNumber(20364031)
	toBlock := geth_rpc.BlockNumber(20364204)

	parameters := thirdparty.ActivityFetchParameters{
		Address:   address, // vitalik.eth
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Incoming,
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
	}

	history, err := alchemyActivityClient.FetchActivity(context.Background(), api_common.MainnetChainID, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 5, len(history.Items))
}

func TestFetchHistoryOutgoing(t *testing.T) {
	alchemyActivityClient := setupAlchemyActivityClient(t)

	address := geth_common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	// Expecting these transfers:
	// https://etherscan.io/advanced-filter?age=2024-07-22%7e2024-07-22&fadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e
	fromBlock := geth_rpc.BlockNumber(20364031)
	toBlock := geth_rpc.BlockNumber(20364204)

	parameters := thirdparty.ActivityFetchParameters{
		Address:   address, // vitalik.eth
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Outgoing,
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
	}

	history, err := alchemyActivityClient.FetchActivity(context.Background(), api_common.MainnetChainID, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 6, len(history.Items))
}
