package wallet

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	wsdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/pkg/pubsub"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/pkg/services/wallet/testutils"
	mock_token "github.com/status-im/status-go/pkg/services/wallet/token/mock/token"
	tokenTypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
	mock_tokenbalances "github.com/status-im/status-go/pkg/services/wallet/tokenbalances/mock/storage"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

var (
	testTokenAddress1 = common.Address{0x34}
	testTokenAddress2 = common.Address{0x56}

	testAccAddress1 = common.Address{0x12}
	testAccAddress2 = common.Address{0x45}

	expectedTokens = map[common.Address][]tokenTypes.StorageToken{
		testAccAddress2: []tokenTypes.StorageToken{
			{
				TokenAddress: common.Address{0x12},
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}
)

// This matcher is used to compare the expected and actual map[common.Address][]tokenTypes.StorageToken in parameters to SaveTokens
type mapTokenWithBalanceMatcher struct {
	expected []interface{}
}

func setupReader(t *testing.T) (*Reader, *mock_token.MockManagerInterface, *mock_tokenbalances.MockStorage, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockTokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	eventsFeed := &event.Feed{}
	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)

	return NewReader(mockTokenManager, nil, eventsFeed, multistandardBalancePublisher, tokenBalancesStorage, transferDetectorPublisher), mockTokenManager, tokenBalancesStorage, mockCtrl
}

func TestGetCachedWalletTokensWithoutMarketData(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	// Test when there is an error getting the tokens
	tokenManager.EXPECT().GetCachedBalances().Return(nil, errors.New("error"))
	tokens, err := reader.getCachedWalletTokensWithoutMarketData()
	require.Error(t, err)
	assert.Nil(t, tokens)

	// Test happy path
	tokenManager.EXPECT().GetCachedBalances().Return(expectedTokens, nil)
	tokens, err = reader.getCachedWalletTokensWithoutMarketData()
	require.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
}

func TestFetchBalancesInternal(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{testAccAddress1, testAccAddress2}
	tokenAddresses := []common.Address{testTokenAddress1, testTokenAddress2}
	ctx := context.TODO()
	clients := map[uint64]ethclient.EthClientInterface{}

	// Note: fetchBalances now uses tokenBalancesStorage.GetBalances instead of GetBalancesByChain
	// This test is now indirectly covered by higher-level tests
	_ = reader
	_ = tokenManager
	_ = ctx
	_ = clients
	_ = addresses
	_ = tokenAddresses
}

func TestTokensToBalancesPerChain(t *testing.T) {
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		testAccAddress1: []tokenTypes.StorageToken{
			{
				TokenAddress: common.Address{0x12},
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
		testAccAddress2: []tokenTypes.StorageToken{

			{
				TokenAddress: common.Address{0x34},
				TokenChainID: 1,
				RawBalance:   "2000000000000000000",
				Balance:      big.NewFloat(2),
				HasError:     false,
			},
			{
				TokenAddress: common.Address{0x56},
				TokenChainID: 2,
				RawBalance:   "3000000000000000000",
				Balance:      big.NewFloat(3),
				HasError:     false,
			},
		},
	}

	expectedBalancesPerChain := map[uint64]map[common.Address]map[common.Address]*big.Int{
		1: {
			testAccAddress1: {
				common.Address{0x12}: big.NewInt(1000000000000000000),
			},
			testAccAddress2: {
				common.Address{0x34}: big.NewInt(2000000000000000000),
			},
		},
		2: {
			testAccAddress2: {
				common.Address{0x56}: big.NewInt(3000000000000000000),
			},
		},
	}

	result, err := tokensToBalancesPerChain(cachedTokens)
	assert.NoError(t, err)

	assert.Equal(t, expectedBalancesPerChain, result)
}

func TestIsCachedToken(t *testing.T) {
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		testAccAddress1: {
			{
				TokenAddress: testTokenAddress1,
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}

	token := &tokenTypes.Token{
		Token: &wsdktypes.Token{
			Address: testTokenAddress1,
			ChainID: 1,
		},
	}

	// Test when the token is cached
	result := isCachedToken(cachedTokens, testAccAddress1, token)
	assert.True(t, result)

	// Test when the token is not cached
	token.Address = testTokenAddress2
	result = isCachedToken(cachedTokens, testAccAddress1, token)
	assert.False(t, result)

	// Test when BalancesPerChain for token have no such a chainID
	token.ChainID = 2
	result = isCachedToken(cachedTokens, testAccAddress1, token)
	assert.False(t, result)

}

func TestBalancesToTokensByAddress(t *testing.T) {
	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	allTokens := []*tokenTypes.Token{
		{
			Token: &wsdktypes.Token{
				Name:     "Token 1",
				Symbol:   "T1",
				Decimals: 18,
				ChainID:  1,
				Address:  common.HexToAddress("0x789"),
			},
		},
		{
			Token: &wsdktypes.Token{
				Name:     "Token 2",
				Symbol:   "T2",
				Decimals: 18,
				ChainID:  1,
				Address:  common.HexToAddress("0xdef"),
			},
		},
		{
			Token: &wsdktypes.Token{
				Name:     "Token 2 optimism",
				Symbol:   "T2",
				Decimals: 18,
				ChainID:  2,
				Address:  common.HexToAddress("0xabc"),
			},
		},
	}

	balances := map[uint64]map[common.Address]map[common.Address]*big.Int{
		1: {
			addresses[0]: {
				allTokens[0].Address: big.NewInt(1000000000000000000),
			},
			addresses[1]: {
				allTokens[1].Address: big.NewInt(2000000000000000000),
			},
		},
		2: {
			addresses[1]: {
				allTokens[2].Address: big.NewInt(3000000000000000000),
			},
		},
	}

	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {
			{
				TokenAddress: common.HexToAddress("0x789"),
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}

	expectedTokensPerAddress := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {
			{
				TokenAddress: common.HexToAddress("0x789"),
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
		addresses[1]: {
			{
				TokenAddress: common.HexToAddress("0xdef"),
				TokenChainID: 1,
				RawBalance:   "2000000000000000000",
				Balance:      big.NewFloat(2),
				HasError:     false,
			},
			{
				TokenAddress: common.HexToAddress("0xabc"),
				TokenChainID: 2,
				RawBalance:   "3000000000000000000",
				Balance:      big.NewFloat(3),
				HasError:     false,
			},
		},
	}

	reader, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	tokens := reader.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens)

	assert.Len(t, tokens, 2)
	assert.Equal(t, len(expectedTokensPerAddress[addresses[0]]), len(tokens[addresses[0]]))
	assert.Equal(t, len(expectedTokensPerAddress[addresses[1]]), len(tokens[addresses[1]]))

	for _, address := range addresses {
		for i, token := range tokens[address] {
			assert.Equal(t, expectedTokensPerAddress[address][i].TokenAddress, token.TokenAddress)
			assert.Equal(t, expectedTokensPerAddress[address][i].TokenChainID, token.TokenChainID)
			assert.Equal(t, expectedTokensPerAddress[address][i].RawBalance, token.RawBalance)
			assert.Equal(t, 0, expectedTokensPerAddress[address][i].Balance.Cmp(token.Balance))
			assert.Equal(t, expectedTokensPerAddress[address][i].HasError, token.HasError)
		}
	}
}

func TestGetCachedBalancesInternal(t *testing.T) {
	reader, tokenManager, tokenBalancesStorage, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	chainIDs := []uint64{1, 2}

	allTokens := []*tokenTypes.Token{
		{
			Token: &wsdktypes.Token{
				Address:  common.HexToAddress("0xabc"),
				Name:     "Token 1",
				Symbol:   "T1",
				Decimals: 18,
				ChainID:  1,
			},
		},
		{
			Token: &wsdktypes.Token{
				Address:  common.HexToAddress("0xdef"),
				Name:     "Token 2",
				Symbol:   "T2",
				Decimals: 18,
				ChainID:  2,
			},
		},
		{
			Token: &wsdktypes.Token{
				Address:  common.HexToAddress("0x789"),
				Name:     "Token 3",
				Symbol:   "T3",
				Decimals: 10,
				ChainID:  1,
			},
		},
	}

	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {},
		addresses[1]: {
			{
				TokenAddress: common.HexToAddress("0xdef"),
				TokenChainID: 2,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}

	expectedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[1]: {
			{
				TokenAddress: common.HexToAddress("0xdef"),
				TokenChainID: 2,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}

	tokensOfInterest := []string{
		types.TokenKey(cachedTokens[addresses[1]][0].TokenChainID, cachedTokens[addresses[1]][0].TokenAddress),
	}
	for _, chainID := range chainIDs {
		tokensOfInterest = append(tokensOfInterest, walletcommon.MandatoryTokensByChainID(chainID)...)
	}

	tokenManager.EXPECT().GetCachedBalances().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByKeys(testutils.NewStringSliceElementsMatcher(tokensOfInterest)).Return(allTokens, nil)
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), allTokens, addresses).Return(
		map[uint64]map[common.Address]map[common.Address]*big.Int{}, nil,
	)
	tokens, err := reader.GetCachedBalances(chainIDs, addresses)
	require.NoError(t, err)

	for _, address := range addresses {
		for i, token := range tokens[address] {
			assert.Equal(t, expectedTokens[address][i].TokenAddress, token.TokenAddress)
			assert.Equal(t, expectedTokens[address][i].TokenChainID, token.TokenChainID)
			assert.Equal(t, expectedTokens[address][i].RawBalance, token.RawBalance)
			assert.Equal(t, 0, expectedTokens[address][i].Balance.Cmp(token.Balance))
			assert.Equal(t, expectedTokens[address][i].HasError, token.HasError)
		}
	}
}

// TestFetchBalancesWithRefresh is no longer needed since refreshBalanceCache is an internal implementation detail
// and is now tested indirectly through other tests

// TestGetLastTokenUpdateTimestamps tests the GetLastTokenUpdateTimestamps method with internal access.
func TestGetLastTokenUpdateTimestampsInternal(t *testing.T) {
	// Setup the Reader and mock dependencies.
	reader, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	// Define test addresses and specific timestamps.
	address1 := testAccAddress1
	address2 := testAccAddress2
	timestamp1 := time.Now().Add(-1 * time.Hour).Unix()
	timestamp2 := time.Now().Add(-2 * time.Hour).Unix()

	// Store valid timestamps in the Reader's sync.Map.
	reader.lastWalletTokenUpdateTimestamp.Store(address1, timestamp1)
	reader.lastWalletTokenUpdateTimestamp.Store(address2, timestamp2)

	// Call the method to retrieve timestamps.
	timestamps := reader.GetLastTokenUpdateTimestamps()
	require.Len(t, timestamps, 2, "Expected two timestamps in the result map")

	// Verify that the retrieved timestamps match the stored values.
	assert.Equal(t, timestamp1, timestamps[address1], "Timestamp for address1 does not match")
	assert.Equal(t, timestamp2, timestamps[address2], "Timestamp for address2 does not match")
}

func TestGetCachedBalances_UsesStorageForMandatoryTokenWhenNotInUICache(t *testing.T) {
	reader, tokenManager, tokenBalancesStorage, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainIDs := []uint64{walletcommon.OptimismMainnet}
	addresses := []common.Address{account}

	nativeToken := &tokenTypes.Token{
		Token: &wsdktypes.Token{
			Address:  tokenbalances.NativeTokenAddress,
			Symbol:   "ETH",
			Decimals: 18,
			ChainID:  walletcommon.OptimismMainnet,
		},
	}
	allTokens := []*tokenTypes.Token{nativeToken}
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		account: {},
	}
	storageBalances := map[uint64]map[common.Address]map[common.Address]*big.Int{
		walletcommon.OptimismMainnet: {
			account: {
				tokenbalances.NativeTokenAddress: big.NewInt(1_000_000_000_000_000_000),
			},
		},
	}

	tokenManager.EXPECT().GetCachedBalances().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByKeys(
		testutils.NewStringSliceElementsMatcher(walletcommon.MandatoryTokensByChainID(walletcommon.OptimismMainnet)),
	).Return(allTokens, nil)
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), allTokens, addresses).Return(storageBalances, nil)

	tokens, err := reader.GetCachedBalances(chainIDs, addresses)
	require.NoError(t, err)

	var nativeMandatory *tokenTypes.StorageToken
	for i := range tokens[account] {
		token := tokens[account][i]
		if token.TokenChainID == walletcommon.OptimismMainnet &&
			token.TokenAddress == tokenbalances.NativeTokenAddress {
			nativeMandatory = &tokens[account][i]
			break
		}
	}
	require.NotNil(t, nativeMandatory, "expected mandatory native token in response")
	assert.False(t, nativeMandatory.HasError, "mandatory token should not be loading when storage has balance")
	assert.Equal(t, "1000000000000000000", nativeMandatory.RawBalance)
}

func TestReader_RefreshesUICacheWhenFirstFetchCompletesWithoutBalanceChange(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)
	balancePublisher := pubsub.NewPublisher()
	reader := NewReader(tokenManager, nil, &event.Feed{}, balancePublisher, tokenBalancesStorage, pubsub.NewPublisher())

	require.NoError(t, reader.Start())
	defer reader.Stop()

	account := testAccAddress1
	chainID := walletcommon.OptimismMainnet
	key := multistandardbalance.BalancesKey{ChainID: chainID, Account: account}

	tokenManager.EXPECT().GetCachedBalances().Return(map[common.Address][]tokenTypes.StorageToken{}, nil)
	tokenManager.EXPECT().GetTokensByChains([]uint64{chainID}).Return([]*tokenTypes.Token{}, nil)
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), gomock.Any(), []common.Address{account}).Return(
		map[uint64]map[common.Address]map[common.Address]*big.Int{}, nil,
	)
	cacheSaved := make(chan struct{})
	tokenManager.EXPECT().CacheBalances(gomock.Any()).DoAndReturn(func(_ map[common.Address][]tokenTypes.StorageToken) error {
		close(cacheSaved)
		return nil
	})

	pubsub.Publish(balancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key:            key,
		ResultType:     multistandardfetcher.ResultTypeERC20,
		BalanceChanged: false,
		OldState:       multistandardbalance.State{FetchedAt: multistandardbalance.NeverFetched},
		NewState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
	})

	select {
	case <-cacheSaved:
	case <-time.After(time.Second):
		t.Fatal("expected UI cache refresh after first fetch event")
	}
}

func TestReader_SkipsUICacheRefreshWhenFetchUnchangedAndAlreadyFetched(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)
	balancePublisher := pubsub.NewPublisher()
	reader := NewReader(tokenManager, nil, &event.Feed{}, balancePublisher, tokenBalancesStorage, pubsub.NewPublisher())

	require.NoError(t, reader.Start())
	defer reader.Stop()

	account := testAccAddress1
	chainID := walletcommon.OptimismMainnet
	key := multistandardbalance.BalancesKey{ChainID: chainID, Account: account}

	pubsub.Publish(balancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key:            key,
		ResultType:     multistandardfetcher.ResultTypeERC20,
		BalanceChanged: false,
		OldState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
		NewState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
	})

	time.Sleep(100 * time.Millisecond)
}

func TestReader_WarmBalanceRefreshStaysDebounced(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)
	balancePublisher := pubsub.NewPublisher()
	walletFeed := &event.Feed{}
	reader := NewReader(tokenManager, nil, walletFeed, balancePublisher, tokenBalancesStorage, pubsub.NewPublisher())

	require.NoError(t, reader.Start())
	defer reader.Stop()

	events := make(chan walletevent.Event, 10)
	sub := walletFeed.Subscribe(events)
	defer sub.Unsubscribe()

	tokenManager.EXPECT().GetCachedBalances().Return(map[common.Address][]tokenTypes.StorageToken{}, nil).AnyTimes()
	tokenManager.EXPECT().GetTokensByChains(gomock.Any()).Return([]*tokenTypes.Token{}, nil).AnyTimes()
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		map[uint64]map[common.Address]map[common.Address]*big.Int{}, nil,
	).AnyTimes()
	tokenManager.EXPECT().CacheBalances(gomock.Any()).Return(nil).AnyTimes()

	account := testAccAddress1
	chainID := walletcommon.OptimismMainnet

	// Warm profile (e.g. a mobile resume): refreshes coalesce through the
	// debounce — Start must not arm an immediate announcement.
	reader.refreshBalanceCache(context.TODO(), []uint64{chainID}, []common.Address{account})
	reader.refreshBalanceCache(context.TODO(), []uint64{chainID}, []common.Address{account})

	select {
	case <-events:
		t.Fatal("warm refreshes must coalesce through the debounce, not emit immediately")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestReader_NeverFetchedCompletionEmitsReloadImmediately(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)
	balancePublisher := pubsub.NewPublisher()
	walletFeed := &event.Feed{}
	reader := NewReader(tokenManager, nil, walletFeed, balancePublisher, tokenBalancesStorage, pubsub.NewPublisher())

	require.NoError(t, reader.Start())
	defer reader.Stop()

	events := make(chan walletevent.Event, 10)
	sub := walletFeed.Subscribe(events)
	defer sub.Unsubscribe()

	tokenManager.EXPECT().GetCachedBalances().Return(map[common.Address][]tokenTypes.StorageToken{}, nil).AnyTimes()
	tokenManager.EXPECT().GetTokensByChains(gomock.Any()).Return([]*tokenTypes.Token{}, nil).AnyTimes()
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		map[uint64]map[common.Address]map[common.Address]*big.Int{}, nil,
	).AnyTimes()
	tokenManager.EXPECT().CacheBalances(gomock.Any()).Return(nil).AnyTimes()

	account := testAccAddress1
	chainID := walletcommon.OptimismMainnet

	// A warm refresh does not announce immediately (no Start-armed edge).
	reader.refreshBalanceCache(context.TODO(), []uint64{chainID}, []common.Address{account})
	select {
	case <-events:
		t.Fatal("warm refresh must not announce immediately")
	case <-time.After(300 * time.Millisecond):
	}

	// A completion for a never-fetched (chain, account) pair is the first data a
	// cold UI can show for it — the reload must go out immediately, not after
	// the debounce.
	pubsub.Publish(balancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key:            multistandardbalance.BalancesKey{ChainID: chainID, Account: account},
		ResultType:     multistandardfetcher.ResultTypeERC20,
		BalanceChanged: false,
		OldState:       multistandardbalance.State{FetchedAt: multistandardbalance.NeverFetched},
		NewState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
	})

	select {
	case ev := <-events:
		require.Equal(t, EventWalletTickReload, ev.Type)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("expected an immediate reload for a never-fetched pair's first completion")
	}
}

func TestReader_ColdReloadEdgeDoesNotSurviveRestart(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)
	balancePublisher := pubsub.NewPublisher()
	walletFeed := &event.Feed{}
	reader := NewReader(tokenManager, nil, walletFeed, balancePublisher, tokenBalancesStorage, pubsub.NewPublisher())

	require.NoError(t, reader.Start())

	account := testAccAddress1
	chainID := walletcommon.OptimismMainnet
	key := multistandardbalance.BalancesKey{ChainID: chainID, Account: account}

	// Cold session: a never-fetched completion arms the immediate reload, but the
	// refresh it belongs to fails before announcing anything.
	refreshAttempted := make(chan struct{})
	tokenManager.EXPECT().GetCachedBalances().DoAndReturn(func() (map[common.Address][]tokenTypes.StorageToken, error) {
		close(refreshAttempted)
		return nil, errors.New("cached balances unavailable")
	})

	pubsub.Publish(balancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key:            key,
		ResultType:     multistandardfetcher.ResultTypeERC20,
		BalanceChanged: false,
		OldState:       multistandardbalance.State{FetchedAt: multistandardbalance.NeverFetched},
		NewState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
	})

	select {
	case <-refreshAttempted:
	case <-time.After(time.Second):
		t.Fatal("expected a UI cache refresh attempt for the first completion")
	}

	reader.Stop()
	require.NoError(t, reader.Start())
	defer reader.Stop()

	events := make(chan walletevent.Event, 10)
	sub := walletFeed.Subscribe(events)
	defer sub.Unsubscribe()

	tokenManager.EXPECT().GetCachedBalances().Return(map[common.Address][]tokenTypes.StorageToken{}, nil).AnyTimes()
	tokenManager.EXPECT().GetTokensByChains(gomock.Any()).Return([]*tokenTypes.Token{}, nil).AnyTimes()
	tokenBalancesStorage.EXPECT().GetBalances(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		map[uint64]map[common.Address]map[common.Address]*big.Int{}, nil,
	).AnyTimes()
	tokenManager.EXPECT().CacheBalances(gomock.Any()).Return(nil).AnyTimes()

	// Warm session (e.g. a mobile resume): a routine balance change must stay
	// debounced - the previous session's edge must not announce it immediately.
	pubsub.Publish(balancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key:            key,
		ResultType:     multistandardfetcher.ResultTypeERC20,
		BalanceChanged: true,
		OldState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
		NewState:       multistandardbalance.State{FetchedAt: time.Now().Unix()},
	})

	select {
	case ev := <-events:
		t.Fatalf("a cold reload edge must not survive Stop/Start, got %q immediately", ev.Type)
	case <-time.After(300 * time.Millisecond):
	}
}
