package token

import (
	"context"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/contracts/snt"
	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdb"
	communitytoken "github.com/status-im/status-go/internal/protocol/communities/token"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	protocolsqlite "github.com/status-im/status-go/internal/protocol/sqlite"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/accounts/accountsevent"
	"github.com/status-im/status-go/pkg/services/communitytokens/communitytokensdatabase"
	"github.com/status-im/status-go/pkg/services/networks"
	network_mock "github.com/status-im/status-go/pkg/services/networks/mock"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

type addressTokenMap = map[common.Address]*tokentypes.Token
type storeMap = map[uint64]addressTokenMap

func setupTestTokenDB(t *testing.T) (*Manager, func()) {
	appDb, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	walletDb, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
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

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)

	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	accountsPublisher := pubsub.NewPublisher()

	networkManager := networks.NewManager(appDB, nil)
	require.NotNil(t, networkManager)
	require.NoError(t, networkManager.InitEmbeddedNetworks(nil))

	config := rpc.ClientConfig{
		NetworkManager: networkManager,
	}
	rpcClient, _ := rpc.NewClient(config)

	mockCtrl := gomock.NewController(t)
	nm := network_mock.NewMockManagerInterface(mockCtrl)
	nm.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: walletcommon.EthereumMainnet},
		{ChainID: walletcommon.OptimismMainnet},
		{ChainID: walletcommon.ArbitrumMainnet},
		{ChainID: walletcommon.BaseMainnet},
		{ChainID: walletcommon.BSCMainnet},
	}, nil).AnyTimes()
	nm.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	nm.EXPECT().GetTestNetworksEnabled().Return(false, nil).AnyTimes()

	manager, err := NewTokenManager(walletDB, rpcClient, nil, nm, appDB, nil, nil, accountsPublisher,
		accountsDB, 1*time.Hour, 1*time.Hour)
	require.NoError(t, err)

	lastTokensUpdate := time.Time{}

	activeChains, err := getEnabledChains(nm)
	require.NoError(t, err)
	tokensManager, err := setUpTokenListsManager(manager, walletDB, activeChains, lastTokensUpdate, 1*time.Hour, 1*time.Hour)
	require.NoError(t, err)
	manager.tokensManager = tokensManager

	return manager, accountsPublisher, func() {
		mockCtrl.Finish()
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
	// Discovered community tokens are annotated with a best-effort privileges
	// level: CommunityLevel when the community-tokens DB has no information.
	expectedCommunityToken := *communityToken
	communityLevel := int(communitytoken.CommunityLevel)
	expectedCommunityToken.PrivilegesLevel = &communityLevel
	require.Equal(t, expectedCommunityToken, *rst[0])
}

func TestCommunityTokensPrivilegesAndSoulbound(t *testing.T) {
	appDb, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	defer func() { require.NoError(t, appDb.Close()) }()
	// community_tokens lives in the protocol migrations
	require.NoError(t, protocolsqlite.Migrate(appDb))

	walletDb, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	defer func() { require.NoError(t, walletDb.Close()) }()

	manager := &Manager{
		walletDB:          walletDb,
		communityTokensDB: communitytokensdatabase.NewCommunityTokensDatabase(appDb),
	}

	addMintedToken := func(address string, tokenType protobuf.CommunityTokenType, transferable bool, level communitytoken.PrivilegesLevel) {
		_, err := appDb.Exec(`INSERT INTO community_tokens (community_id, address, type, name, symbol, description, supply_str,
			infinite_supply, transferable, remote_self_destruct, chain_id, deploy_state, image_base64, decimals, deployer, privileges_level)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"community_id", address, tokenType, "name-"+address, "SYM", "desc", "1",
			false, transferable, false, 777, communitytoken.Deployed, "", 0, "0xDEP", level)
		require.NoError(t, err)
	}

	ownerAddress := common.Address{1}.Hex()
	masterAddress := common.Address{2}.Hex()
	assetAddress := common.Address{3}.Hex()
	// the owner token is transferable by design (that is how community ownership moves)
	addMintedToken(ownerAddress, protobuf.CommunityTokenType_ERC721, true, communitytoken.OwnerLevel)
	// the TMaster token is non-transferable (soulbound)
	addMintedToken(masterAddress, protobuf.CommunityTokenType_ERC721, false, communitytoken.MasterLevel)
	// a regular community asset
	addMintedToken(assetAddress, protobuf.CommunityTokenType_ERC20, true, communitytoken.CommunityLevel)

	// A community token merely discovered on-chain: present in the wallet tokens
	// table, absent from community_tokens — there is no metadata to annotate with.
	discovered := &tokentypes.Token{
		Token: &types.Token{
			Address:  common.Address{4},
			Name:     "Discovered",
			Symbol:   "DIS",
			Decimals: 12,
			ChainID:  777,
		},
		CommunityData: &tokentypes.CommunityData{ID: "other_community"},
	}
	upsertCommunityToken(t, discovered, manager)

	rst, err := manager.GetCustoms(true)
	require.NoError(t, err)
	require.Equal(t, 4, len(rst))

	byAddress := make(map[common.Address]*tokentypes.Token)
	for _, token := range rst {
		byAddress[token.Address] = token
	}

	requireLevel := func(token *tokentypes.Token, level communitytoken.PrivilegesLevel) {
		require.NotNil(t, token.PrivilegesLevel)
		require.Equal(t, int(level), *token.PrivilegesLevel)
	}

	owner := byAddress[common.HexToAddress(ownerAddress)]
	require.NotNil(t, owner)
	requireLevel(owner, communitytoken.OwnerLevel)
	require.False(t, owner.Soulbound)

	master := byAddress[common.HexToAddress(masterAddress)]
	require.NotNil(t, master)
	requireLevel(master, communitytoken.MasterLevel)
	require.True(t, master.Soulbound)

	asset := byAddress[common.HexToAddress(assetAddress)]
	require.NotNil(t, asset)
	requireLevel(asset, communitytoken.CommunityLevel)
	require.False(t, asset.Soulbound)

	// Absent metadata: the community-tokens DB exists but has no row for the
	// discovered token, so the privileges level stays unknown (nil) and the
	// token is not marked soulbound.
	discoveredResult := byAddress[discovered.Address]
	require.NotNil(t, discoveredResult)
	require.Nil(t, discoveredResult.PrivilegesLevel)
	require.False(t, discoveredResult.Soulbound)
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
			if slices.Contains(walletcommon.SkippedTokenKeys(), token.Key()) {
				require.Equal(t, 0, numOfOccurrences, "token %s is in skipped token keys", token.Key())
				continue
			}
			require.Equal(t, 1, numOfOccurrences, "token %s appears %d times in allTokens", token.Key(), numOfOccurrences)
		}
	}
}

func TestGetTokensOfInterestForActiveNetworksMode_AddsCrossChainAndMandatoryTokens(t *testing.T) {
	manager, _, stop := setupTestTokenManager(t)
	defer stop()

	require.NoError(t, manager.Start(context.Background()))
	defer manager.Stop()

	testnetMode, err := manager.settings.GetTestNetworksEnabled()
	require.NoError(t, err)
	require.False(t, testnetMode)

	// Cache some tokens that are not among mandatory tokens and have cross chain id
	err = manager.CacheBalances(map[common.Address][]tokentypes.StorageToken{
		common.HexToAddress("0x0000000000000000000000000000000000000001"): {
			{
				TokenAddress: common.HexToAddress("0x514910771af9ca656af840dff83e8264ecf986ca"), // chainlink
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
			{
				TokenAddress: common.HexToAddress("0x6fd9d7ad17242c41f7131d257212c54a0e816691"), // uniswap
				TokenChainID: 10,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	})
	require.NoError(t, err)

	// token keys from cache, all of them have cross chain id
	cachedTokenKeys := []string{
		types.TokenKey(1, common.HexToAddress("0x514910771af9ca656af840dff83e8264ecf986ca")),
		types.TokenKey(10, common.HexToAddress("0x6fd9d7ad17242c41f7131d257212c54a0e816691")),
	}

	activeNetworks, err := manager.networkManager.GetActiveNetworks()
	require.NoError(t, err)

	// mandatory token keys for the active networks mode, all of them have cross chain id
	mandatoryTokenKeysForMode := []string{}
	for _, tokenKey := range walletcommon.MandatoryTokens() {
		chainID, _, ok := types.ChainAndAddressFromTokenKey(tokenKey)
		if !ok {
			continue
		}
		if walletcommon.ChainID(chainID).IsMainnet() == testnetMode {
			continue
		}
		if !slices.ContainsFunc(activeNetworks, func(network *params.Network) bool {
			return network.ChainID == chainID
		}) {
			continue
		}
		mandatoryTokenKeysForMode = append(mandatoryTokenKeysForMode, tokenKey)
	}

	tokens, err := manager.GetTokensByKeys(append(cachedTokenKeys, mandatoryTokenKeysForMode...))
	require.NoError(t, err)
	require.NotEmpty(t, tokens)
	require.Equal(t, len(cachedTokenKeys)+len(mandatoryTokenKeysForMode), len(tokens))

	expectedCrossChainIDs := make(map[string]int)
	for _, token := range tokens {
		expectedCrossChainIDs[token.CrossChainID] = 0
	}

	tokensForActiveNetworksMode, err := manager.GetTokensOfInterestForActiveNetworksMode()
	require.NoError(t, err)
	require.NotEmpty(t, tokensForActiveNetworksMode)

	for _, token := range tokensForActiveNetworksMode {
		_, ok := expectedCrossChainIDs[token.CrossChainID]
		require.True(t, ok, "token with cross chain id %s is not in the expected cross chain ids", token.CrossChainID)
		expectedCrossChainIDs[token.CrossChainID]++
	}

	const (
		bscNativeTokenCrossChainID = "bsc-native"   // nolint: gosec
		bscUsdcTokenCrossChainID   = "usd-coin-bsc" // nolint: gosec
	)
	for crossChainID, count := range expectedCrossChainIDs {
		switch crossChainID {
		case bscNativeTokenCrossChainID, bscUsdcTokenCrossChainID:
			require.Equal(t, count, 1, "expected 1 token with cross chain id %s, got %d", crossChainID, count)
		default:
			require.Greater(t, count, 1, "expected more than 1 token with cross chain id %s, got %d", crossChainID, count)
		}
	}
}
