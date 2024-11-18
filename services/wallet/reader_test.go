package wallet

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/rpc/chain"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/token"
	mock_balance_persistence "github.com/status-im/status-go/services/wallet/token/mock/balance_persistence"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
)

var (
	testTokenAddress1 = common.Address{0x34}
	testTokenAddress2 = common.Address{0x56}

	testAccAddress1 = common.Address{0x12}
	testAccAddress2 = common.Address{0x45}

	expectedTokens = map[common.Address][]token.StorageToken{
		testAccAddress1: []token.StorageToken{
			{
				Token: token.Token{
					Address:  testTokenAddress1,
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: nil,
			},
		},
		testAccAddress2: []token.StorageToken{
			{
				Token: token.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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

// This matcher is used to compare the expected and actual map[common.Address][]token.StorageToken in parameters to SaveTokens
type mapTokenWithBalanceMatcher struct {
	expected []interface{}
}

func (m mapTokenWithBalanceMatcher) Matches(x interface{}) bool {
	actual, ok := x.(map[common.Address][]token.StorageToken)
	if !ok {
		return false
	}

	if len(m.expected) != len(actual) {
		return false
	}

	expected := m.expected[0].(map[common.Address][]token.StorageToken)

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
func testChainBalancesEqual(t *testing.T, expected, actual token.ChainBalance) {
	assert.Equal(t, expected.RawBalance, actual.RawBalance)
	assert.Equal(t, 0, expected.Balance.Cmp(actual.Balance))
	assert.Equal(t, expected.Address, actual.Address)
	assert.Equal(t, expected.ChainID, actual.ChainID)
	assert.Equal(t, expected.HasError, actual.HasError)
	assert.Equal(t, expected.Balance1DayAgo, actual.Balance1DayAgo)
}

func testBalancePerChainEqual(t *testing.T, expected, actual map[uint64]token.ChainBalance) {
	assert.Len(t, actual, len(expected))
	for chainID, expectedBalance := range expected {
		actualBalance, ok := actual[chainID]
		assert.True(t, ok)
		testChainBalancesEqual(t, expectedBalance, actualBalance)
	}
}

func setupReader(t *testing.T) (*Reader, *mock_token.MockManagerInterface, *mock_balance_persistence.MockTokenBalancesStorage, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockTokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenBalanceStorage := mock_balance_persistence.NewMockTokenBalancesStorage(mockCtrl)
	eventsFeed := &event.Feed{}

	return NewReader(mockTokenManager, nil, tokenBalanceStorage, eventsFeed), mockTokenManager, tokenBalanceStorage, mockCtrl
}

func TestGetCachedWalletTokensWithoutMarketData(t *testing.T) {
	reader, _, tokenPersistence, mockCtrl := setupReader(t)
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

func TestIsBalanceCacheValid(t *testing.T) {
	reader, _, tokenPersistence, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	// Mock the cache to be valid
	addresses := []common.Address{testAccAddress1, testAccAddress2}
	reader.balanceRefreshed()
	reader.updateTokenUpdateTimestamp([]common.Address{testAccAddress1, testAccAddress2})
	tokenPersistence.EXPECT().GetTokens().Return(expectedTokens, nil)
	valid := reader.isBalanceCacheValid(addresses)
	assert.True(t, valid)

	// Mock the cache to be invalid
	reader.invalidateBalanceCache()
	valid = reader.isBalanceCacheValid(addresses)
	assert.False(t, valid)

	// Make cached tokens not contain all the addresses
	reader.balanceRefreshed()
	cachedTokens := map[common.Address][]token.StorageToken{
		testAccAddress1: expectedTokens[testAccAddress1],
	}
	tokenPersistence.EXPECT().GetTokens().Return(cachedTokens, nil)
	valid = reader.isBalanceCacheValid(addresses)
	assert.False(t, valid)

	// Some timestamps are not updated
	reader.balanceRefreshed()
	reader.lastWalletTokenUpdateTimestamp = sync.Map{}
	reader.updateTokenUpdateTimestamp([]common.Address{testAccAddress1})
	cachedTokens = expectedTokens
	tokenPersistence.EXPECT().GetTokens().AnyTimes().Return(cachedTokens, nil)
	assert.False(t, valid)
}

func TestTokensCachedForAddresses(t *testing.T) {
	reader, _, persistence, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{testAccAddress1, testAccAddress2}

	// Test when the cached tokens do not contain all the addresses
	cachedTokens := map[common.Address][]token.StorageToken{
		testAccAddress1: expectedTokens[testAccAddress1],
	}
	persistence.EXPECT().GetTokens().Return(cachedTokens, nil)

	result := reader.tokensCachedForAddresses(addresses)
	assert.False(t, result)

	// Test when the cached tokens contain all the addresses
	cachedTokens = expectedTokens
	persistence.EXPECT().GetTokens().Return(cachedTokens, nil)

	result = reader.tokensCachedForAddresses(addresses)
	assert.True(t, result)
}

func TestFetchBalancesInternal(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{testAccAddress1, testAccAddress2}
	tokenAddresses := []common.Address{testTokenAddress1, testTokenAddress2}
	ctx := context.TODO()
	clients := map[uint64]chain.ClientInterface{}

	// Test when there is an error getting the tokens
	tokenManager.EXPECT().GetBalancesByChain(ctx, clients, addresses, tokenAddresses).Return(nil, errors.New("error"))
	balances, err := reader.fetchBalances(ctx, clients, addresses, tokenAddresses)
	require.Error(t, err)
	assert.Nil(t, balances)

	// Test happy path
	expectedBalances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			testAccAddress2: {
				testTokenAddress2: (*hexutil.Big)(big.NewInt(1)),
			},
		},
	}
	tokenManager.EXPECT().GetBalancesByChain(ctx, clients, addresses, tokenAddresses).Return(expectedBalances, nil)
	balances, err = reader.fetchBalances(ctx, clients, addresses, tokenAddresses)
	require.NoError(t, err)
	assert.Equal(t, balances, expectedBalances)
}

func TestTokensToBalancesPerChain(t *testing.T) {
	cachedTokens := map[common.Address][]token.StorageToken{
		testAccAddress1: []token.StorageToken{
			{
				Token: token.Token{
					Address:  testTokenAddress1,
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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
				Token: token.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: nil, // Skip this token
			},
		},
		testAccAddress2: []token.StorageToken{
			{
				Token: token.Token{
					Address:  testTokenAddress2,
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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

	result := tokensToBalancesPerChain(cachedTokens)

	assert.Equal(t, expectedBalancesPerChain, result)
}

func TestGetBalance1DayAgo(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	address := common.Address{0x12}
	chainID := uint64(1)
	symbol := "T1"
	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()

	// Test happy path
	expectedBalance := big.NewInt(1000000000000000000)
	tokenManager.EXPECT().GetTokenHistoricalBalance(address, chainID, symbol, dayAgoTimestamp).Return(expectedBalance, nil)

	balance1DayAgo, err := reader.getBalance1DayAgo(&token.ChainBalance{
		ChainID: chainID,
		Address: address,
	}, dayAgoTimestamp, symbol, address)

	require.NoError(t, err)
	assert.Equal(t, expectedBalance, balance1DayAgo)

	// Test error
	tokenManager.EXPECT().GetTokenHistoricalBalance(address, chainID, symbol, dayAgoTimestamp).Return(nil, errors.New("error"))
	balance1DayAgo, err = reader.getBalance1DayAgo(&token.ChainBalance{
		ChainID: chainID,
		Address: address,
	}, dayAgoTimestamp, symbol, address)

	require.Error(t, err)
	assert.Nil(t, balance1DayAgo)
}

func TestToChainBalance(t *testing.T) {
	balances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		1: {
			common.Address{0x12}: {
				common.Address{0x34}: (*hexutil.Big)(big.NewInt(1000000000000000000)),
			},
		},
	}
	tok := &token.Token{
		ChainID:  1,
		Address:  common.Address{0x34},
		Symbol:   "T1",
		Decimals: 18,
	}
	address := common.Address{0x12}
	decimals := uint(18)
	cachedTokens := map[common.Address][]token.StorageToken{
		common.Address{0x12}: {
			{
				Token: token.Token{
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
	expectedChainBalance := &token.ChainBalance{
		RawBalance:     "1000000000000000000",
		Balance:        expectedBalance,
		Balance1DayAgo: "0",
		Address:        common.Address{0x34},
		ChainID:        1,
		HasError:       hasError,
	}

	chainBalance := toChainBalance(balances, tok, address, decimals, cachedTokens, hasError, false)
	testChainBalancesEqual(t, *expectedChainBalance, *chainBalance)

	// Test when the token is not visible
	emptyCachedTokens := map[common.Address][]token.StorageToken{}
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
	cachedTokens := map[common.Address][]token.StorageToken{
		common.Address{0x12}: {
			{
				Token: token.Token{
					Address:  common.Address{0x34},
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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

	tokens := []*token.Token{
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
	cachedTokens := map[common.Address][]token.StorageToken{
		address: {
			{
				Token: token.Token{
					Address:  common.Address{0x34},
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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

	clientConnectionPerChain := map[uint64]bool{
		1: true,
		2: false,
	}
	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()

	expectedBalancesPerChain := map[uint64]token.ChainBalance{
		1: {
			RawBalance:     "1000000000000000000",
			Balance:        big.NewFloat(1),
			Balance1DayAgo: "0",
			Address:        common.Address{0x34},
			ChainID:        1,
			HasError:       false,
		},
		2: {
			RawBalance:     "2000000000000000000",
			Balance:        big.NewFloat(2),
			Balance1DayAgo: "0",
			Address:        common.Address{0x56},
			ChainID:        2,
			HasError:       true,
		},
	}

	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	tokenManager.EXPECT().GetTokenHistoricalBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil, errors.New("error"))
	result := reader.createBalancePerChainPerSymbol(address, balances, tokens, cachedTokens, clientConnectionPerChain, dayAgoTimestamp)

	assert.Len(t, result, 2)
	testBalancePerChainEqual(t, expectedBalancesPerChain, result)
}

func TestCreateBalancePerChainPerSymbolWithMissingBalance(t *testing.T) {
	address := common.Address{0x12}
	tokens := []*token.Token{
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

	clientConnectionPerChain := map[uint64]bool{
		1: true,
		2: false,
	}

	dayAgoTimestamp := time.Now().Add(-24 * time.Hour).Unix()
	emptyCachedTokens := map[common.Address][]token.StorageToken{}
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

	expectedBalancesPerChain := map[uint64]token.ChainBalance{
		2: {
			RawBalance:     "1000000000000000000",
			Balance:        big.NewFloat(1),
			Balance1DayAgo: "1",
			Address:        common.Address{0x56},
			ChainID:        2,
			HasError:       true,
		},
	}

	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	tokenManager.EXPECT().GetTokenHistoricalBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(big.NewInt(1), nil)
	result := reader.createBalancePerChainPerSymbol(address, oneBalanceMissing, tokens, emptyCachedTokens, clientConnectionPerChain, dayAgoTimestamp)
	assert.Len(t, result, 1)
	testBalancePerChainEqual(t, expectedBalancesPerChain, result)
}

func TestBalancesToTokensByAddress(t *testing.T) {
	connectedPerChain := map[uint64]bool{
		1: true,
		2: true,
	}

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	allTokens := []*token.Token{
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

	cachedTokens := map[common.Address][]token.StorageToken{
		addresses[0]: {
			{
				Token: token.Token{
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
					Verified: true,
					Address:  common.HexToAddress("0x789"),
					ChainID:  1,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
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

	expectedTokensPerAddress := map[common.Address][]token.StorageToken{
		addresses[0]: {
			{
				Token: token.Token{
					Name:     "Token 1",
					Symbol:   "T1",
					Decimals: 18,
					Verified: true,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					1: {
						RawBalance:     "1000000000000000000",
						Balance:        big.NewFloat(1),
						Address:        common.HexToAddress("0x789"),
						ChainID:        1,
						HasError:       false,
						Balance1DayAgo: "0",
					},
				},
			},
		},
		addresses[1]: {
			{
				Token: token.Token{
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
					Verified: true,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					1: {
						RawBalance:     "2000000000000000000",
						Balance:        big.NewFloat(2),
						Address:        common.HexToAddress("0xdef"),
						ChainID:        1,
						HasError:       false,
						Balance1DayAgo: "0",
					},
					2: {
						RawBalance:     "3000000000000000000",
						Balance:        big.NewFloat(3),
						Address:        common.HexToAddress("0xabc"),
						ChainID:        2,
						HasError:       false,
						Balance1DayAgo: "0",
					},
				},
			},
		},
	}

	reader, tokenManager, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	tokenManager.EXPECT().GetTokenHistoricalBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	tokens := reader.balancesToTokensByAddress(connectedPerChain, addresses, allTokens, balances, cachedTokens)

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

func TestGetCachedBalances(t *testing.T) {
	reader, tokenManager, persistence, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	mockClientIface1 := mock_client.NewMockClientInterface(mockCtrl)
	mockClientIface2 := mock_client.NewMockClientInterface(mockCtrl)
	clients := map[uint64]chain.ClientInterface{
		1: mockClientIface1,
		2: mockClientIface2,
	}

	allTokens := []*token.Token{
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

	cachedTokens := map[common.Address][]token.StorageToken{
		addresses[0]: {
			{
				Token: token.Token{
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
				Token: token.Token{
					Address:  common.HexToAddress("0xdef"),
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
					ChainID:  2,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					2: {
						RawBalance:     "1000000000000000000",
						Balance:        big.NewFloat(1),
						Balance1DayAgo: "0",
						Address:        common.HexToAddress("0xdef"),
						ChainID:        2,
						HasError:       false,
					},
				},
			},
		},
	}

	expectedTokens := map[common.Address][]token.StorageToken{
		addresses[1]: {
			{
				Token: token.Token{
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					2: {
						RawBalance:     "1000000000000000000",
						Balance:        big.NewFloat(1),
						Balance1DayAgo: "0",
						Address:        common.HexToAddress("0xdef"),
						ChainID:        2,
						HasError:       false,
					},
				},
			},
		},
	}

	persistence.EXPECT().GetTokens().Return(cachedTokens, nil)
	expectedChains := []uint64{1, 2}
	tokenManager.EXPECT().GetTokensByChainIDs(testutils.NewUint64SliceMatcher(expectedChains)).Return(allTokens, nil)
	tokenManager.EXPECT().GetTokenHistoricalBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mockClientIface1.EXPECT().IsConnected().Return(true)
	mockClientIface2.EXPECT().IsConnected().Return(true)
	tokens, err := reader.GetCachedBalances(clients, addresses)
	require.NoError(t, err)

	for _, address := range addresses {
		for i, token := range tokens[address] {
			assert.Equal(t, expectedTokens[address][i].Token, token.Token)
			testBalancePerChainEqual(t, expectedTokens[address][i].BalancesPerChain, token.BalancesPerChain)
		}
	}
}

func TestFetchBalances(t *testing.T) {
	reader, tokenManager, persistence, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	mockClientIface1 := mock_client.NewMockClientInterface(mockCtrl)
	mockClientIface2 := mock_client.NewMockClientInterface(mockCtrl)
	clients := map[uint64]chain.ClientInterface{
		1: mockClientIface1,
		2: mockClientIface2,
	}

	allTokens := []*token.Token{
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

	cachedTokens := map[common.Address][]token.StorageToken{
		addresses[0]: {
			{
				Token: token.Token{
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
				Token: token.Token{
					Address:  common.HexToAddress("0xdef"),
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
					ChainID:  2,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					2: {
						RawBalance:     "1000000000000000000",
						Balance:        big.NewFloat(1),
						Balance1DayAgo: "0",
						Address:        common.HexToAddress("0xdef"),
						ChainID:        2,
						HasError:       false,
					},
				},
			},
		},
	}

	expectedTokens := map[common.Address][]token.StorageToken{
		addresses[1]: {
			{
				Token: token.Token{
					Name:     "Token 2",
					Symbol:   "T2",
					Decimals: 18,
				},
				BalancesPerChain: map[uint64]token.ChainBalance{
					2: {
						RawBalance:     "2000000000000000000",
						Balance:        big.NewFloat(2),
						Balance1DayAgo: "0",
						Address:        common.HexToAddress("0xdef"),
						ChainID:        2,
						HasError:       false,
					},
				},
			},
		},
	}

	persistence.EXPECT().GetTokens().Times(2).Return(cachedTokens, nil)
	// Verify that proper tokens are saved
	persistence.EXPECT().SaveTokens(newMapTokenWithBalanceMatcher([]interface{}{expectedTokens})).Return(nil)
	expectedChains := []uint64{1, 2}
	tokenManager.EXPECT().GetTokensByChainIDs(testutils.NewUint64SliceMatcher(expectedChains)).Return(allTokens, nil)
	tokenManager.EXPECT().GetTokenHistoricalBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	expectedBalances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		2: {
			addresses[1]: {
				allTokens[1].Address: (*hexutil.Big)(big.NewInt(2000000000000000000)),
			},
		},
	}

	tokenAddresses := getTokenAddresses(allTokens)
	tokenManager.EXPECT().GetBalancesByChain(context.TODO(), clients, addresses, testutils.NewAddressSliceMatcher(tokenAddresses)).Return(expectedBalances, nil)
	mockClientIface1.EXPECT().IsConnected().Return(true)
	mockClientIface2.EXPECT().IsConnected().Return(true)
	tokens, err := reader.FetchBalances(context.TODO(), clients, addresses)
	require.NoError(t, err)

	for address, tokenList := range tokens {
		for i, token := range tokenList {
			assert.Equal(t, expectedTokens[address][i].Token, token.Token)
			testBalancePerChainEqual(t, expectedTokens[address][i].BalancesPerChain, token.BalancesPerChain)
		}
	}

	require.True(t, reader.isBalanceCacheValid(addresses))
}

func TestReaderRestart(t *testing.T) {
	reader, _, _, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	err := reader.Start()
	require.NoError(t, err)
	require.NotNil(t, reader.walletEventsWatcher)
	previousWalletEventsWatcher := reader.walletEventsWatcher

	err = reader.Restart()
	require.NoError(t, err)
	require.NotNil(t, reader.walletEventsWatcher)
	require.NotEqual(t, previousWalletEventsWatcher, reader.walletEventsWatcher)
}

func TestFetchOrGetCachedWalletBalances(t *testing.T) {
	// Test the behavior of FetchOrGetCachedWalletBalances when fetching new balances fails.
	// This focuses on the error handling path where the function should return the cached balances as a fallback.
	// We don't explicitly test the contents of fetched or cached balances here, as those
	// are covered in other dedicated tests. The main goal is to ensure the correct flow of
	// execution and data retrieval in this specific failure scenario.

	reader, _, tokenPersistence, mockCtrl := setupReader(t)
	defer mockCtrl.Finish()

	reader.invalidateBalanceCache()

	tokenPersistence.EXPECT().GetTokens().Return(nil, errors.New("error")).AnyTimes()

	clients := map[uint64]chain.ClientInterface{}
	addresses := []common.Address{}

	_, err := reader.FetchOrGetCachedWalletBalances(context.TODO(), clients, addresses, false)
	require.Error(t, err)
}

// TestGetLastTokenUpdateTimestamps tests the GetLastTokenUpdateTimestamps method.
func TestGetLastTokenUpdateTimestamps(t *testing.T) {
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
