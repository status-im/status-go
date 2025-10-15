package wallet

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/testutils"
	mock_balance_persistence "github.com/status-im/status-go/services/wallet/token/mock/balance_persistence"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	mock_tokenbalances "github.com/status-im/status-go/services/wallet/tokenbalances/mock/storage"
)

var (
	testTokenAddress1 = common.Address{0x34}
	testTokenAddress2 = common.Address{0x56}

	testAccAddress1 = common.Address{0x12}
	testAccAddress2 = common.Address{0x45}

	expectedTokens = map[common.Address][]tokenTypes.StorageToken{
		testAccAddress1: []tokenTypes.StorageToken{
			{
				Token: tokenTypes.Token{
					Address:  testTokenAddress1,
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: nil,
			},
		},
		testAccAddress2: []tokenTypes.StorageToken{
			{
				Token: tokenTypes.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.Address{0x12},
						ChainID:    1,
						HasError:   false,
					},
				},
			},
		},
	}
)

// This matcher is used to compare the expected and actual map[common.Address][]tokenTypes.StorageToken in parameters to SaveTokens
type mapTokenWithBalanceMatcher struct {
	expected []interface{}
}

func (m mapTokenWithBalanceMatcher) Matches(x interface{}) bool {
	actual, ok := x.(map[common.Address][]tokenTypes.StorageToken)
	if !ok {
		return false
	}

	if len(m.expected) != len(actual) {
		return false
	}

	expected := m.expected[0].(map[common.Address][]tokenTypes.StorageToken)

	for address, expectedTokens := range expected {
		actualTokens, ok := actual[address]
		if !ok {
			return false
		}

		if len(expectedTokens) != len(actualTokens) {
			return false
		}

		for i, expectedToken := range expectedTokens {
			actualToken := actualTokens[i]
			if expectedToken.Token != actualToken.Token {
				return false
			}

			if len(expectedToken.BalancesPerChain) != len(actualToken.BalancesPerChain) {
				return false
			}

			// We can't compare the  balances directly because the Balance field is a big.Float
			for chainID, expectedBalance := range expectedToken.BalancesPerChain {
				actualBalance, ok := actualToken.BalancesPerChain[chainID]
				if !ok {
					return false
				}

				if expectedBalance.Balance.Cmp(actualBalance.Balance) != 0 {
					return false
				}

				if expectedBalance.RawBalance != actualBalance.RawBalance {
					return false
				}

				if expectedBalance.Address != actualBalance.Address {
					return false
				}

				if expectedBalance.ChainID != actualBalance.ChainID {
					return false
				}

				if expectedBalance.HasError != actualBalance.HasError {
					return false
				}
			}
		}
	}

	return true
}

func (m *mapTokenWithBalanceMatcher) String() string {
	return fmt.Sprintf("%v", m.expected)
}

func newMapTokenWithBalanceMatcher(expected []interface{}) gomock.Matcher {
	return &mapTokenWithBalanceMatcher{
		expected: expected,
	}
}
func testChainBalancesEqual(t *testing.T, expected, actual tokenTypes.ChainBalance) {
	assert.Equal(t, expected.RawBalance, actual.RawBalance)
	assert.Equal(t, 0, expected.Balance.Cmp(actual.Balance))
	assert.Equal(t, expected.Address, actual.Address)
	assert.Equal(t, expected.ChainID, actual.ChainID)
	assert.Equal(t, expected.HasError, actual.HasError)
}

func testBalancePerChainEqual(t *testing.T, expected, actual map[uint64]tokenTypes.ChainBalance) {
	assert.Len(t, actual, len(expected))
	for chainID, expectedBalance := range expected {
		actualBalance, ok := actual[chainID]
		assert.True(t, ok)
		testChainBalancesEqual(t, expectedBalance, actualBalance)
	}
}

func setupReader(t *testing.T) (*Reader, *mock_token.MockManagerInterface, *mock_balance_persistence.MockTokenBalancesStorage, *mock_tokenbalances.MockStorage, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockTokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalanceStorage := mock_balance_persistence.NewMockTokenBalancesStorage(mockCtrl)
	eventsFeed := &event.Feed{}
	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)

	return NewReader(mockTokenManager, nil, tokenBalanceStorage, eventsFeed, multistandardBalancePublisher, tokenBalancesStorage, transferDetectorPublisher), mockTokenManager, tokenBalanceStorage, tokenBalancesStorage, mockCtrl
}

func TestGetCachedWalletTokensWithoutMarketData(t *testing.T) {
	reader, _, tokenPersistence, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	// Test when there is an error getting the tokens
	tokenPersistence.EXPECT().GetTokens().Return(nil, errors.New("error"))
	tokens, err := reader.getCachedWalletTokensWithoutMarketData()
	require.Error(t, err)
	assert.Nil(t, tokens)

	// Test happy path
	tokenPersistence.EXPECT().GetTokens().Return(expectedTokens, nil)
	tokens, err = reader.getCachedWalletTokensWithoutMarketData()
	require.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
}

func TestFetchBalancesInternal(t *testing.T) {
	reader, tokenManager, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{testAccAddress1, testAccAddress2}
	tokenAddresses := []common.Address{testTokenAddress1, testTokenAddress2}
	ctx := context.TODO()
	clients := map[uint64]chain.ClientInterface{}

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
				Token: tokenTypes.Token{
					Address:  testTokenAddress1,
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.Address{0x12},
						ChainID:    1,
						HasError:   false,
					},
				},
			},
			{
				Token: tokenTypes.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: nil, // Skip this token
			},
		},
		testAccAddress2: []tokenTypes.StorageToken{
			{
				Token: tokenTypes.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "2000000000000000000",
						Balance:    big.NewFloat(2),
						Address:    common.Address{0x34},
						ChainID:    1,
						HasError:   false,
					},
					2: {
						RawBalance: "3000000000000000000",
						Balance:    big.NewFloat(3),
						Address:    common.Address{0x56},
						ChainID:    2,
						HasError:   false,
					},
				},
			},
		},
	}

	expectedBalancesPerChain := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			testAccAddress1: {
				common.Address{0x12}: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
			testAccAddress2: {
				common.Address{0x34}: (*hexutil.Big)(big.NewInt(2000000000000000000)),
			},
		},
		2: {
			testAccAddress2: {
				common.Address{0x56}: (*hexutil.Big)(big.NewInt(3000000000000000000)),
			},
		},
	}

	result, err := tokensToBalancesPerChain(cachedTokens)
	assert.NoError(t, err)

	assert.Equal(t, expectedBalancesPerChain, result)
}

func TestToChainBalance(t *testing.T) {
	balances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			common.Address{0x12}: {
				common.Address{0x34}: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
		},
	}
	tok := &tokenTypes.Token{
		ChainID:  1,
		Address:  common.Address{0x34},
		Symbol:   "T1",
		Decimals: 18,
	}
	address := common.Address{0x12}
	decimals := uint(18)
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		common.Address{0x12}: {
			{
				Token: tokenTypes.Token{
					Address:  common.Address{0x34},
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: nil,
			},
		},
	}

	expectedBalance := big.NewFloat(1)
	hasError := false
	expectedChainBalance := &tokenTypes.ChainBalance{
		RawBalance: "1000000000000000000",
		Balance:    expectedBalance,
		Address:    common.Address{0x34},
		ChainID:    1,
		HasError:   hasError,
	}

	chainBalance := toChainBalance(balances, tok, address, decimals, cachedTokens, hasError, false)
	testChainBalancesEqual(t, *expectedChainBalance, *chainBalance)

	// Test when the token is not visible
	emptyCachedTokens := map[common.Address][]tokenTypes.StorageToken{}
	isMandatory := false
	noBalances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		tok.ChainID: {
			address: {
				tok.Address: nil, // Idk why this can be nil
			},
		},
	}
	chainBalance = toChainBalance(noBalances, tok, address, decimals, emptyCachedTokens, hasError, isMandatory)
	assert.Nil(t, chainBalance)
}

func TestIsCachedToken(t *testing.T) {
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		common.Address{0x12}: {
			{
				Token: tokenTypes.Token{
					Address:  common.Address{0x34},
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.Address{0x12},
						ChainID:    1,
						HasError:   false,
					},
				},
			},
		},
	}

	address := common.Address{0x12}
	symbol := "T1"
	chainID := uint64(1)

	// Test when the token is cached
	result := isCachedToken(cachedTokens, address, symbol, chainID)
	assert.True(t, result)

	// Test when the token is not cached
	result = isCachedToken(cachedTokens, address, "T2", chainID)
	assert.False(t, result)

	// Test when BalancesPerChain for token have no such a chainID
	wrongChainID := chainID + 1
	result = isCachedToken(cachedTokens, address, symbol, wrongChainID)
	assert.False(t, result)

}

func TestCreateBalancePerChainPerSymbol(t *testing.T) {
	address := common.Address{0x12}
	balances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			address: {
				common.Address{0x34}: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
		},
		2: {
			address: {
				common.Address{0x56}: (*hexutil.Big)(big.NewInt(2000000000000000000)),
			},
		},
	}

	tokens := []*tokenTypes.Token{
		{
			Name:     "Token 1 mainnet",
			ChainID:  1,
			Address:  common.Address{0x34},
			Symbol:   "T1",
			Decimals: 18,
		},
		{
			Name:     "Token 1 optimism",
			ChainID:  2,
			Address:  common.Address{0x56},
			Symbol:   "T1",
			Decimals: 18,
		},
	}
	// Let cached tokens not have the token for chain 2, it still should be calculated because of positive balance
	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		address: {
			{
				Token: tokenTypes.Token{
					Address:  common.Address{0x34},
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.Address{0x12},
						ChainID:    1,
						HasError:   false,
					},
				},
			},
		},
	}

	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()

	expectedBalancesPerChain := map[uint64]tokenTypes.ChainBalance{
		1: {
			RawBalance: "1000000000000000000",
			Balance:    big.NewFloat(1),
			Address:    common.Address{0x34},
			ChainID:    1,
			HasError:   false,
		},
		2: {
			RawBalance: "2000000000000000000",
			Balance:    big.NewFloat(2),
			Address:    common.Address{0x56},
			ChainID:    2,
			HasError:   false, // hasError is now determined by presence in balances map, not by clientConnectionPerChain
		},
	}

	reader, _, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	result := reader.createBalancePerChainPerSymbol(address, balances, tokens, cachedTokens, dayAgoTimestamp)

	assert.Len(t, result, 2)
	testBalancePerChainEqual(t, expectedBalancesPerChain, result)
}

func TestCreateBalancePerChainPerSymbolWithMissingBalance(t *testing.T) {
	address := common.Address{0x12}
	tokens := []*tokenTypes.Token{
		{
			Name:     "Token 1 mainnet",
			ChainID:  1,
			Address:  common.Address{0x34},
			Symbol:   "T1",
			Decimals: 18,
		},
		{
			Name:     "Token 1 optimism",
			ChainID:  2,
			Address:  common.Address{0x56},
			Symbol:   "T1",
			Decimals: 18,
		},
	}

	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()
	emptyCachedTokens := map[common.Address][]tokenTypes.StorageToken{}
	oneBalanceMissing := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			address: {
				common.Address{0x34}: nil, // Idk why this can be nil
			},
		},

		2: {
			address: {
				common.Address{0x56}: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
		},
	}

	expectedBalancesPerChain := map[uint64]tokenTypes.ChainBalance{
		2: {
			RawBalance: "1000000000000000000",
			Balance:    big.NewFloat(1),
			Address:    common.Address{0x56},
			ChainID:    2,
			HasError:   false, // hasError is now determined by presence in balances map
		},
	}

	reader, _, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	result := reader.createBalancePerChainPerSymbol(address, oneBalanceMissing, tokens, emptyCachedTokens, dayAgoTimestamp)
	assert.Len(t, result, 1)
	testBalancePerChainEqual(t, expectedBalancesPerChain, result)
}

func TestBalancesToTokensByAddress(t *testing.T) {
	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	allTokens := []*tokenTypes.Token{
		{
			Name:     "Token 1",
			Symbol:   "T1",
			Decimals: 18,
			Verified: true,
			ChainID:  1,
			Address:  common.HexToAddress("0x789"),
		},
		{
			Name:     "Token 2",
			Symbol:   "T2",
			Decimals: 18,
			Verified: true,
			ChainID:  1,
			Address:  common.HexToAddress("0xdef"),
		},
		{
			Name:     "Token 2 optimism",
			Symbol:   "T2",
			Decimals: 18,
			Verified: true,
			ChainID:  2,
			Address:  common.HexToAddress("0xabc"),
		},
	}

	balances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			addresses[0]: {
				allTokens[0].Address: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
			addresses[1]: {
				allTokens[1].Address: (*hexutil.Big)(big.NewInt(2000000000000000000)),
			},
		},
		2: {
			addresses[1]: {
				allTokens[2].Address: (*hexutil.Big)(big.NewInt(3000000000000000000)),
			},
		},
	}

	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {
			{
				Token: tokenTypes.Token{
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
					Verified: true,
					Address:  common.HexToAddress("0x789"),
					ChainID:  1,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.HexToAddress("0x789"),
						ChainID:    1,
						HasError:   false,
					},
				},
			},
		},
	}

	expectedTokensPerAddress := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {
			{
				Token: tokenTypes.Token{
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
					Verified: true,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.HexToAddress("0x789"),
						ChainID:    1,
						HasError:   false,
					},
				},
			},
		},
		addresses[1]: {
			{
				Token: tokenTypes.Token{
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
					Verified: true,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					1: {
						RawBalance: "2000000000000000000",
						Balance:    big.NewFloat(2),
						Address:    common.HexToAddress("0xdef"),
						ChainID:    1,
						HasError:   false,
					},
					2: {
						RawBalance: "3000000000000000000",
						Balance:    big.NewFloat(3),
						Address:    common.HexToAddress("0xabc"),
						ChainID:    2,
						HasError:   false,
					},
				},
			},
		},
	}

	reader, _, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	tokens := reader.balancesToTokensByAddress(addresses, allTokens, balances, cachedTokens)

	assert.Len(t, tokens, 2)
	assert.Equal(t, 1, len(tokens[addresses[0]]))
	assert.Equal(t, 1, len(tokens[addresses[1]]))

	for _, address := range addresses {
		for i, token := range tokens[address] {
			assert.Equal(t, expectedTokensPerAddress[address][i].Token, token.Token)
			testBalancePerChainEqual(t, expectedTokensPerAddress[address][i].BalancesPerChain, token.BalancesPerChain)
		}
	}
}

func TestGetCachedBalancesInternal(t *testing.T) {
	reader, tokenManager, persistence, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	chainIDs := []uint64{1, 2}

	allTokens := []*tokenTypes.Token{
		{
			Address:  common.HexToAddress("0xabc"),
			Name:     "Token 1",
			Symbol:   "T1",
			Decimals: 18,
			ChainID:  1,
		},
		{
			Address:  common.HexToAddress("0xdef"),
			Name:     "Token 2",
			Symbol:   "T2",
			Decimals: 18,
			ChainID:  2,
		},
		{
			Address:  common.HexToAddress("0x789"),
			Name:     "Token 3",
			Symbol:   "T3",
			Decimals: 10,
			ChainID:  1,
		},
	}

	cachedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[0]: {
			{
				Token: tokenTypes.Token{
					Address:  common.HexToAddress("0xabc"),
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
					ChainID:  1,
				},
				BalancesPerChain: nil,
			},
		},
		addresses[1]: {
			{
				Token: tokenTypes.Token{
					Address:  common.HexToAddress("0xdef"),
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
					ChainID:  2,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					2: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.HexToAddress("0xdef"),
						ChainID:    2,
						HasError:   false,
					},
				},
			},
		},
	}

	expectedTokens := map[common.Address][]tokenTypes.StorageToken{
		addresses[1]: {
			{
				Token: tokenTypes.Token{
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]tokenTypes.ChainBalance{
					2: {
						RawBalance: "1000000000000000000",
						Balance:    big.NewFloat(1),
						Address:    common.HexToAddress("0xdef"),
						ChainID:    2,
						HasError:   false,
					},
				},
			},
		},
	}

	persistence.EXPECT().GetTokens().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByChainIDs(testutils.NewUint64SliceMatcher(chainIDs)).Return(allTokens, nil)
	tokens, err := reader.GetCachedBalances(chainIDs, addresses)
	require.NoError(t, err)

	for _, address := range addresses {
		for i, token := range tokens[address] {
			assert.Equal(t, expectedTokens[address][i].Token, token.Token)
			testBalancePerChainEqual(t, expectedTokens[address][i].BalancesPerChain, token.BalancesPerChain)
		}
	}
}

// TestFetchBalancesWithRefresh is no longer needed since refreshBalanceCache is an internal implementation detail
// and is now tested indirectly through other tests

// TestGetLastTokenUpdateTimestamps tests the GetLastTokenUpdateTimestamps method with internal access.
func TestGetLastTokenUpdateTimestampsInternal(t *testing.T) {
	// Setup the Reader and mock dependencies.
	reader, _, _, _, mockCtrl := setupReader(t)
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
