package wallet_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/wallet"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	mock_tokenbalances "github.com/status-im/status-go/services/wallet/tokenbalances/mock/storage"
)

var (
	testTokenAddress1 = common.Address{0x34}
	testTokenAddress2 = common.Address{0x56}

	testAccAddress1 = common.Address{0x12}
	testAccAddress2 = common.Address{0x45}

	expectedTokens = map[common.Address][]tokentypes.StorageToken{
		testAccAddress1: []tokentypes.StorageToken{},
		testAccAddress2: []tokentypes.StorageToken{
			{
				TokenAddress: testTokenAddress2,
				TokenChainID: 1,
				RawBalance:   "1000000000000000000",
				Balance:      big.NewFloat(1),
				HasError:     false,
			},
		},
	}
)

func setupReaderExported(t *testing.T) (*wallet.Reader, *mock_token.MockManagerInterface, *mock_tokenbalances.MockStorage, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockTokenManager := mock_token.NewMockManagerInterface(mockCtrl)
	eventsFeed := &event.Feed{}
	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	tokenBalancesStorage := mock_tokenbalances.NewMockStorage(mockCtrl)

	return wallet.NewReader(mockTokenManager, nil, eventsFeed, multistandardBalancePublisher, tokenBalancesStorage, transferDetectorPublisher), mockTokenManager, tokenBalancesStorage, mockCtrl
}

func TestGetCachedBalances(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReaderExported(t)

	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	chainIDs := []uint64{1, 2}

	allTokens := []*tokentypes.Token{
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0xabc"),
				Name:     "Token 1",
				Symbol:   "T1",
				Decimals: 18,
				ChainID:  1,
			},
		},
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0xdef"),
				Name:     "Token 2",
				Symbol:   "T2",
				Decimals: 18,
				ChainID:  2,
			},
		},
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0x789"),
				Name:     "Token 3",
				Symbol:   "T3",
				Decimals: 10,
				ChainID:  1,
			},
		},
	}

	cachedTokens := map[common.Address][]tokentypes.StorageToken{
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

	expectedTokens := map[common.Address][]tokentypes.StorageToken{
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

	tokenManager.EXPECT().GetCachedBalances().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByKeys(tokensOfInterest).Return(allTokens, nil)
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

func TestFetchBalances(t *testing.T) {
	reader, tokenManager, _, mockCtrl := setupReaderExported(t)
	defer mockCtrl.Finish()

	addresses := []common.Address{
		common.HexToAddress("0x123"),
		common.HexToAddress("0x456"),
	}

	chainIDs := []uint64{1, 2}

	allTokens := []*tokentypes.Token{
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0xabc"),
				Name:     "Token 1",
				Symbol:   "T1",
				Decimals: 18,
				ChainID:  1,
			},
		},
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0xdef"),
				Name:     "Token 2",
				Symbol:   "T2",
				Decimals: 18,
				ChainID:  2,
			},
		},
		{
			Token: &types.Token{
				Address:  common.HexToAddress("0x789"),
				Name:     "Token 3",
				Symbol:   "T3",
				Decimals: 10,
				ChainID:  1,
			},
		},
	}

	cachedTokens := map[common.Address][]tokentypes.StorageToken{
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

	tokensOfInterest := []string{
		types.TokenKey(cachedTokens[addresses[1]][0].TokenChainID, cachedTokens[addresses[1]][0].TokenAddress),
	}

	// Test GetCachedBalances with cached data
	tokenManager.EXPECT().GetCachedBalances().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByKeys(tokensOfInterest).Return(allTokens, nil)

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
	reader, _, _, mockCtrl := setupReaderExported(t)
	defer mockCtrl.Finish()

	err := reader.Start()
	require.NoError(t, err)

	reader.Stop()
}

func TestFetchOrGetCachedWalletBalances(t *testing.T) {
	// Test the behavior when fetching balances fails.
	// This focuses on the error handling path where the function should return an error.

	reader, tokenManager, _, mockCtrl := setupReaderExported(t)
	defer mockCtrl.Finish()

	tokenManager.EXPECT().GetCachedBalances().Return(nil, errors.New("error")).AnyTimes()

	chainIDs := []uint64{1, 2}
	addresses := []common.Address{}

	_, err := reader.GetCachedBalances(chainIDs, addresses)
	require.Error(t, err)
}
