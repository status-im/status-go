package tokenbalances_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
)

func TestGetBalances_ERC20NeverFetched_MissingTokenNotInMap(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	adapter := tokenbalances.NewStorageMultistandardBalance(storage)

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainID := uint64(8453)
	token := erc20Token(chainID, "0x820C137fA9D5348B4B9B2E229CBeD970E8e7E360")

	balances, err := adapter.GetBalances(context.Background(), []*tokentypes.Token{token}, []common.Address{account})
	require.NoError(t, err)

	_, ok := balances[chainID][account][token.Address]
	assert.False(t, ok, "token should be absent when ERC20 balances were never fetched")
}

// Zero balances are not stored, so a token missing from a fetched map is a zero.
func TestGetBalances_ERC20Fetched_MissingTokenIsZero(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	adapter := tokenbalances.NewStorageMultistandardBalance(storage)

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainID := uint64(8453)
	key := multistandardbalance.BalancesKey{Account: account, ChainID: chainID}

	knownToken := erc20Token(chainID, "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	zeroToken := erc20Token(chainID, "0x820C137fA9D5348B4B9B2E229CBeD970E8e7E360")

	_, _, err := storage.UpdateERC20Balances(context.Background(), key, map[common.Address]*big.Int{
		knownToken.Address: big.NewInt(42),
	}, fetchedState())
	require.NoError(t, err)

	balances, err := adapter.GetBalances(context.Background(), []*tokentypes.Token{knownToken, zeroToken}, []common.Address{account})
	require.NoError(t, err)

	knownBalance, ok := balances[chainID][account][knownToken.Address]
	require.True(t, ok)
	assert.Equal(t, 0, knownBalance.Cmp(big.NewInt(42)))

	zeroBalance, ok := balances[chainID][account][zeroToken.Address]
	require.True(t, ok, "a token missing from a fetched map is present after fetch")
	require.NotNil(t, zeroBalance, "... as a zero balance, not as unknown")
	assert.Equal(t, 0, zeroBalance.Sign())
}

func TestGetBalances_ERC20Fetched_UnansweredTokenIsUnknown(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	adapter := tokenbalances.NewStorageMultistandardBalance(storage)

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainID := uint64(8453)
	key := multistandardbalance.BalancesKey{Account: account, ChainID: chainID}
	unansweredToken := erc20Token(chainID, "0x820C137fA9D5348B4B9B2E229CBeD970E8e7E360")

	_, _, err := storage.UpdateERC20Balances(context.Background(), key, map[common.Address]*big.Int{
		unansweredToken.Address: nil,
	}, fetchedState())
	require.NoError(t, err)

	balances, err := adapter.GetBalances(context.Background(), []*tokentypes.Token{unansweredToken}, []common.Address{account})
	require.NoError(t, err)

	balance, ok := balances[chainID][account][unansweredToken.Address]
	require.True(t, ok, "a token the fetch did not answer for is present after fetch")
	assert.Nil(t, balance, "an unanswered token has no value, it is not a zero balance")
}

func TestGetBalances_NativeNeverFetched_NotInMap(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	adapter := tokenbalances.NewStorageMultistandardBalance(storage)

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainID := uint64(8453)
	nativeToken := nativeToken(chainID)

	balances, err := adapter.GetBalances(context.Background(), []*tokentypes.Token{nativeToken}, []common.Address{account})
	require.NoError(t, err)

	_, ok := balances[chainID][account][tokenbalances.NativeTokenAddress]
	assert.False(t, ok, "native balance should be absent when never fetched")
}

func TestGetBalances_NativeFetched_InMap(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	adapter := tokenbalances.NewStorageMultistandardBalance(storage)

	account := common.HexToAddress("0x2a811F1E11636C144a2A062d3D402245A43D4074")
	chainID := uint64(8453)
	key := multistandardbalance.BalancesKey{Account: account, ChainID: chainID}
	nativeToken := nativeToken(chainID)

	_, _, err := storage.UpdateNativeBalance(context.Background(), key, big.NewInt(1000), fetchedState())
	require.NoError(t, err)

	balances, err := adapter.GetBalances(context.Background(), []*tokentypes.Token{nativeToken}, []common.Address{account})
	require.NoError(t, err)

	nativeBalance, ok := balances[chainID][account][tokenbalances.NativeTokenAddress]
	require.True(t, ok)
	assert.Equal(t, 0, nativeBalance.Cmp(big.NewInt(1000)))
}

func erc20Token(chainID uint64, address string) *tokentypes.Token {
	return &tokentypes.Token{
		Token: &types.Token{
			ChainID: chainID,
			Address: common.HexToAddress(address),
		},
	}
}

func nativeToken(chainID uint64) *tokentypes.Token {
	return &tokentypes.Token{
		Token: &types.Token{
			ChainID: chainID,
			Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
			Symbol:  "ETH",
		},
	}
}

func fetchedState() multistandardbalance.State {
	return multistandardbalance.State{
		AtBlockNumber: big.NewInt(100),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}
}
