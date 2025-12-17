package token

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/internal/contracts/snt"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/pubsub"
	protocolsqlite "github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

type addressTokenMap = map[common.Address]*tokentypes.Token
type storeMap = map[uint64]addressTokenMap

func setupTestTokenDB(t *testing.T) (*Manager, func()) {
	appDb, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDb, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	return &Manager{
			walletDB:             walletDb,
			ethClientGetter:      nil,
			ContractMaker:        nil,
			networkManager:       nil,
			communityTokensDB:    nil,
			communityManager:     nil,
			tokenBalancesStorage: balanceStorage{walletDB: walletDb},
		}, func() {
			require.NoError(t, appDb.Close())
			require.NoError(t, walletDb.Close())
		}
}

func setupTestTokenManager(t *testing.T) (*Manager, *pubsub.Publisher, func()) {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	err = protocolsqlite.Migrate(appDB)
	require.NoError(t, err)

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	accountsPublisher := pubsub.NewPublisher()

	config := rpc.ClientConfig{
		Networks: nil,
		DB:       appDB,
	}
	rpcClient, _ := rpc.NewClient(config)

	nm := network.NewManager(appDB, nil)

	manager, err := NewTokenManager(walletDB, rpcClient, nil, nm, appDB, nil, nil, accountsPublisher,
		accountsDB, 1*time.Hour, 1*time.Hour)
	require.NoError(t, err)

	lastTokensUpdate := time.Time{}

	tokensManager, err := setUpTokenListsManager(manager, walletDB, lastTokensUpdate, 1*time.Hour, 1*time.Hour)
	require.NoError(t, err)
	manager.tokensManager = tokensManager

	return manager, accountsPublisher, func() {
		require.NoError(t, appDB.Close())
		require.NoError(t, walletDB.Close())
	}
}

func upsertCommunityToken(t *testing.T, token *tokentypes.Token, manager *Manager) {
	require.NotNil(t, token.CommunityData)

	err := manager.UpsertCustom(*token)
	require.NoError(t, err)

	// Community ID is only discovered by calling contract, so must be updated manually
	_, err = manager.walletDB.Exec("UPDATE tokens SET community_id = ? WHERE address = ?", token.CommunityData.ID, token.Address)
	require.NoError(t, err)
}

func TestCustoms(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.GetCustoms(false)
	require.NoError(t, err)
	require.Nil(t, rst)

	token := tokentypes.Token{
		Token: &types.Token{
			Address:  common.Address{1},
			Name:     "Zilliqa",
			Symbol:   "ZIL",
			Decimals: 12,
			ChainID:  777,
		},
	}

	err = manager.UpsertCustom(token)
	require.NoError(t, err)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	err = manager.DeleteCustom(777, token.Address)
	require.NoError(t, err)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}

func TestCommunityTokens(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	rst, err := manager.GetCustoms(true)
	require.NoError(t, err)
	require.Nil(t, rst)

	token := tokentypes.Token{
		Token: &types.Token{
			Address:  common.Address{1},
			Name:     "Zilliqa",
			Symbol:   "ZIL",
			Decimals: 12,
			ChainID:  777,
		},
	}

	err = manager.UpsertCustom(token)
	require.NoError(t, err)

	communityToken := &tokentypes.Token{
		Token: &types.Token{
			Address:  common.Address{2},
			Name:     "Communitia",
			Symbol:   "COM",
			Decimals: 12,
			ChainID:  777,
		},
		CommunityData: &tokentypes.CommunityData{
			ID: "random_community_id",
		},
	}

	upsertCommunityToken(t, communityToken, manager)

	rst, err = manager.GetCustoms(false)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, token, *rst[0])

	rst, err = manager.GetCustoms(true)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, *communityToken, *rst[0])
}

func TestMarkAsPreviouslyOwnedToken(t *testing.T) {
	manager, stop := setupTestTokenDB(t)
	defer stop()

	owner := common.HexToAddress("0x1234567890abcdef")
	token := &tokentypes.Token{
		Token: &types.Token{
			Address:  common.HexToAddress("0xabcdef1234567890"),
			Name:     "TestToken",
			Symbol:   "TT",
			Decimals: 18,
			ChainID:  1,
		},
	}

	isFirst, err := manager.MarkAsPreviouslyOwnedToken(nil, owner)
	require.Error(t, err)
	require.False(t, isFirst)

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, common.Address{})
	require.Error(t, err)
	require.False(t, isFirst)

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)
	require.True(t, isFirst)

	// Verify that the token balance was inserted correctly
	var count int
	err = manager.walletDB.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)
	require.False(t, isFirst)

	// Not updated because already exists
	err = manager.walletDB.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	token.ChainID = 2

	isFirst, err = manager.MarkAsPreviouslyOwnedToken(token, owner)
	require.NoError(t, err)

	// Same token on different chains counts as different token
	err = manager.walletDB.QueryRow(`SELECT count(*) FROM token_balances`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.True(t, isFirst)
}

func Test_removeTokenBalanceOnEventAccountRemoved(t *testing.T) {
	manager, accountsPublisher, stop := setupTestTokenManager(t)
	defer stop()

	err := manager.Start(context.Background())
	require.NoError(t, err)

	address := common.HexToAddress("0x1234")

	chainID := uint64(1)
	sntAddress, err := snt.ContractAddress(chainID)
	require.NoError(t, err)

	// Insert balances for address
	marked, err := manager.MarkAsPreviouslyOwnedToken(&tokentypes.Token{
		Token: &types.Token{
			Address:  sntAddress,
			Symbol:   "Dummy",
			Decimals: 18,
			ChainID:  chainID,
		},
	}, address)
	require.NoError(t, err)
	require.True(t, marked)

	tokenByAddress, err := manager.GetPreviouslyOwnedTokens()
	require.NoError(t, err)
	require.Len(t, tokenByAddress, 1)

	// Start service
	err = manager.Start(context.Background())
	require.NoError(t, err)

	// Watching accounts must start before sending event.
	// To avoid running goroutine immediately and let the controller subscribe first,
	// use any delay.
	group := sync.WaitGroup{}
	group.Add(1)
	go func() {
		defer group.Done()
		time.Sleep(1 * time.Millisecond)

		pubsub.Publish(accountsPublisher, accountsevent.AccountsRemovedEvent{
			Accounts: []common.Address{address},
		})

		require.Eventually(t, func() bool {
			tokenByAddress, err := manager.GetPreviouslyOwnedTokens()
			return err == nil && len(tokenByAddress) == 0
		}, 100*time.Millisecond, 10*time.Millisecond)
	}()

	group.Wait()

	// Stop service
	manager.Stop()
}

func Test_tokensListsValidity(t *testing.T) {
	manager, _, stop := setupTestTokenManager(t)
	defer stop()

	err := manager.Start(context.Background())
	require.NoError(t, err)

	allLists, err := manager.GetAllTokenLists()
	require.NoError(t, err)
	require.Greater(t, len(allLists), 0)

	allTokens, err := manager.GetAllTokens()
	require.NoError(t, err)
	require.Greater(t, len(allTokens), 0)

	allTokensForActiveNetworksMode, err := manager.GetTokensForActiveNetworksMode()
	require.NoError(t, err)
	require.Greater(t, len(allTokensForActiveNetworksMode), 0)

	testnetMode, err := manager.settings.GetTestNetworksEnabled()
	require.NoError(t, err)

	for _, token := range allTokensForActiveNetworksMode {
		require.True(t, testnetMode == !walletcommon.ChainID(token.Token.ChainID).IsMainnet())
	}

	// every token from the list should appear in the allTokens only once
	for _, list := range allLists {
		for _, token := range list.Tokens {
			numOfOccurrences := 0
			for _, t := range allTokens {
				if t.Address == token.Address {
					numOfOccurrences++
					break
				}
			}
			require.Equal(t, 1, numOfOccurrences)
		}
	}
}
