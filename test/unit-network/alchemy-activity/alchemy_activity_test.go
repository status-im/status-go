package alchemy_activity_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	geth_common "github.com/ethereum/go-ethereum/common"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/networks"
	alchemymanager "github.com/status-im/status-go/pkg/services/wallet/activityfetcher/alchemy"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
	t_common "github.com/status-im/status-go/test/unit-network/common"
)

func setupAlchemyActivityManager(t *testing.T) *alchemymanager.Manager {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)

	walletConfig := t_common.GetWalletConfigFromEnv()
	defaultNetworks := networks.BuildDefaultNetworks(walletConfig, true)
	networkManager := networks.NewManager(appDB, nil)
	err = networkManager.InitEmbeddedNetworks(defaultNetworks)
	require.NoError(t, err)

	config := rpc.ClientConfig{
		NetworkManager: networkManager,
	}
	rpcClient, err := rpc.NewClient(config)
	require.NoError(t, err)

	ctx := context.Background()
	rpcClient.Start(ctx)
	t.Cleanup(func() { rpcClient.Stop() })

	alchemyEthClientGetter := rpc.NewProviderChainClientGetter(common.SmartProxyAlchemy, rpcClient)

	alchemyClient := alchemy.NewClient(alchemyEthClientGetter)
	alchemyPersistence := alchemy.NewPersistence(walletDB)
	alchemyManager := alchemymanager.NewManager(alchemyClient, alchemyPersistence)

	return alchemyManager
}

func TestFetchHistoryBoth(t *testing.T) {
	alchemyActivityManager := setupAlchemyActivityManager(t)

	address := geth_common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e")
	// Expecting these transfers:
	// https://etherscan.io/advanced-filter?fadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e&tadd=0xa1e277ea6b97effc5b61b3bf5de03f438981247e&age=2024-07-22%7e2024-07-22
	fromBlock := geth_rpc.BlockNumber(20364031)
	toBlock := geth_rpc.BlockNumber(20364204)

	parameters := thirdparty.ActivityFetchParameters{
		Address:   address, // vitalik.eth
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Both,
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
	}

	history, err := alchemyActivityManager.FetchActivity(context.Background(), common.EthereumMainnet, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 6, len(history.Items))
}

func TestFetchHistoryIncoming(t *testing.T) {
	alchemyActivityManager := setupAlchemyActivityManager(t)

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

	history, err := alchemyActivityManager.FetchActivity(context.Background(), common.EthereumMainnet, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 5, len(history.Items))
}

func TestFetchHistoryOutgoing(t *testing.T) {
	alchemyActivityManager := setupAlchemyActivityManager(t)

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

	history, err := alchemyActivityManager.FetchActivity(context.Background(), common.EthereumMainnet, parameters, thirdparty.FetchFromStartCursor, thirdparty.FetchNoLimit)
	require.NoError(t, err)
	require.Equal(t, 5, len(history.Items))
}
