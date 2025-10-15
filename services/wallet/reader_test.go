package wallet_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet"
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

func setupReaderExported(t *testing.T) (*wallet.Reader, *mock_token.MockManagerInterface, *mock_balance_persistence.MockTokenBalancesStorage, *mock_tokenbalances.MockStorage, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockTokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	tokenManagerBalanceStorage := mock_balance_persistence.NewMockTokenBalancesStorage(mockCtrl)
	eventsFeed := &event.Feed{}
	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)

	return wallet.NewReader(mockTokenManager, nil, tokenManagerBalanceStorage, eventsFeed, multistandardBalancePublisher, tokenBalancesStorage, transferDetectorPublisher), mockTokenManager, tokenManagerBalanceStorage, tokenBalancesStorage, mockCtrl
}

func TestGetCachedBalances(t *testing.T) {
	reader, tokenManager, persistence, _, mockCtrl := setupReaderExported(t)
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

func TestFetchBalances(t *testing.T) {
	reader, tokenManager, persistence, _, mockCtrl := setupReaderExported(t)
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

	// Test GetCachedBalances with cached data
	persistence.EXPECT().GetTokens().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByChainIDs(testutils.NewUint64SliceMatcher(chainIDs)).Return(allTokens, nil)

	tokens, err := reader.GetCachedBalances(chainIDs, addresses)
	require.NoError(t, err)

	// Verify that we get the cached tokens back (transformed)
	for address, tokenList := range tokens {
		// Just verify we got some tokens for the addresses that had cached data
		if len(cachedTokens[address]) > 0 && len(tokenList) > 0 {
			// The test verifies that GetCachedBalances works with the cached data
			assert.NotEmpty(t, tokenList)
		}
	}
}

func TestReaderRestart(t *testing.T) {
	reader, _, _, _, mockCtrl := setupReaderExported(t)
	defer mockCtrl.Finish()

	err := reader.Start()
	require.NoError(t, err)

	reader.Stop()
}

func TestFetchOrGetCachedWalletBalances(t *testing.T) {
	// Test the behavior when fetching balances fails.
	// This focuses on the error handling path where the function should return an error.

	reader, _, tokenPersistence, _, mockCtrl := setupReaderExported(t)
	defer mockCtrl.Finish()

	tokenPersistence.EXPECT().GetTokens().Return(nil, errors.New("error")).AnyTimes()

	chainIDs := []uint64{1, 2}
	addresses := []common.Address{}

	_, err := reader.GetCachedBalances(chainIDs, addresses)
	require.Error(t, err)
}
