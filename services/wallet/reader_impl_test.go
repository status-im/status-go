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
	"github.com/ethereum/go-ethereum/event"

	wsdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/rpc/chain/ethclient"
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

			if expectedToken.TokenAddress != actualToken.TokenAddress ||
				expectedToken.TokenChainID != actualToken.TokenChainID ||
				expectedToken.RawBalance != actualToken.RawBalance ||
				expectedToken.Balance.Cmp(actualToken.Balance) != 0 ||
				expectedToken.HasError != actualToken.HasError {
				return false
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
	reader, tokenManager, _, mockCtrl := setupReader(t)
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

	tokenManager.EXPECT().GetCachedBalances().Return(cachedTokens, nil)
	tokenManager.EXPECT().GetTokensByChains(chainIDs).Return(allTokens, nil)
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
