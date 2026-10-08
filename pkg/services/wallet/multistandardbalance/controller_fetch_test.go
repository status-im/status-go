package multistandardbalance

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
)

type countingTokenListProvider struct {
	calls map[uint64]int
}

func (p *countingTokenListProvider) GetTokenContractAddresses(chainID uint64) ([]common.Address, error) {
	p.calls[chainID]++
	return []common.Address{{byte(chainID)}}, nil
}

func TestBuildFetchConfigsReadsTokenListOncePerChain(t *testing.T) {
	provider := &countingTokenListProvider{calls: make(map[uint64]int)}
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), nil, nil, nil, nil, provider, nil, nil, nil, zap.NewNop())

	erc20 := []multistandardfetcher.ResultType{multistandardfetcher.ResultTypeERC20}
	toFetch := make(map[BalancesKey][]multistandardfetcher.ResultType)
	for _, chainID := range []uint64{1, 10} {
		for i := 0; i < 3; i++ {
			toFetch[BalancesKey{Account: common.Address{19: byte(i + 1)}, ChainID: chainID}] = erc20
		}
	}

	configs := c.buildMultiStandardFetcherFetchConfigs(toFetch)

	require.Equal(t, map[uint64]int{1: 1, 10: 1}, provider.calls)
	for _, chainID := range []uint64{1, 10} {
		require.Len(t, configs[chainID].ERC20, 3)
		for _, tokens := range configs[chainID].ERC20 {
			require.Equal(t, []common.Address{{byte(chainID)}}, tokens)
		}
	}
}

type tokenWrapper struct{ address common.Address }

// listBuildingTokenListProvider allocates like the production adapter: one
// wrapper per listed token and append-grown slices, per call.
type listBuildingTokenListProvider struct {
	list []common.Address
}

func (p *listBuildingTokenListProvider) GetTokenContractAddresses(uint64) ([]common.Address, error) {
	tokens := make([]*tokenWrapper, 0)
	for _, address := range p.list {
		tokens = append(tokens, &tokenWrapper{address: address})
	}
	addresses := make([]common.Address, 0)
	for _, token := range tokens {
		addresses = append(addresses, token.address)
	}
	return addresses, nil
}

func BenchmarkBuildFetchConfigs(b *testing.B) {
	provider := &listBuildingTokenListProvider{list: make([]common.Address, 8500)}
	for i := range provider.list {
		provider.list[i] = common.Address{0x70, 18: byte(i >> 8), 19: byte(i)}
	}
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), nil, nil, nil, nil, provider, nil, nil, nil, zap.NewNop())
	toFetch := map[BalancesKey][]multistandardfetcher.ResultType{}
	for i := 0; i < 2; i++ {
		toFetch[BalancesKey{Account: common.Address{19: byte(i + 1)}, ChainID: 1}] = []multistandardfetcher.ResultType{multistandardfetcher.ResultTypeERC20}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = c.buildMultiStandardFetcherFetchConfigs(toFetch)
	}
}

func erc20FetchResult(block int64, account common.Address, balances map[common.Address]*big.Int, failed ...common.Address) multistandardfetcher.FetchResult {
	return multistandardfetcher.FetchResult{
		ResultType: multistandardfetcher.ResultTypeERC20,
		Result: multistandardfetcher.ERC20Result{
			Account:       account,
			Results:       balances,
			Failed:        failed,
			AtBlockNumber: big.NewInt(block),
		},
	}
}

func storedERC20Balances(t *testing.T, c *Controller, account common.Address) ERC20Balances {
	stored, state, err := c.storage.GetERC20Balances(context.Background(), BalancesKey{Account: account, ChainID: 1})
	require.NoError(t, err)
	require.NotEqual(t, NeverFetched, state.FetchedAt)
	return stored
}

func erc20Config(account common.Address, asked ...common.Address) multistandardfetcher.FetchConfig {
	return multistandardfetcher.FetchConfig{ERC20: map[AccountAddress][]ContractAddress{account: asked}}
}

// With zero balances left out of the results, a token the fetch asked for and
// that is missing from the result is a zero unless its call failed.
func TestHandleERC20Result_ZeroAndFailedAcrossFetches(t *testing.T) {
	account := common.Address{19: 1}
	dropsToZero := common.Address{0x70, 19: 1}
	fails := common.Address{0x70, 19: 2}
	ctx := context.Background()
	config := erc20Config(account, dropsToZero, fails)
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), nil, nil, nil, nil, nil, nil, noopLastBlockManager{}, nil, zap.NewNop())

	c.handleFetchResult(ctx, 1, config, erc20FetchResult(1, account, map[common.Address]*big.Int{dropsToZero: big.NewInt(5), fails: big.NewInt(7)}))
	require.Equal(t, int64(5), storedERC20Balances(t, c, account).Balances[dropsToZero].Int64())

	c.handleFetchResult(ctx, 1, config, erc20FetchResult(2, account, map[common.Address]*big.Int{}, fails))
	stored := storedERC20Balances(t, c, account)
	require.NotContains(t, stored.Balances, dropsToZero, "5 -> 0: not stored")
	require.Contains(t, stored.Answered, dropsToZero, "... answered, so it reads 0")
	require.Equal(t, int64(7), stored.Balances[fails].Int64(), "a failed call keeps the last known balance")
}

func TestHandleERC20Result_FailedWithoutKnownBalanceIsUnknown(t *testing.T) {
	account := common.Address{19: 1}
	fails := common.Address{0x70, 19: 2}
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), nil, nil, nil, nil, nil, nil, noopLastBlockManager{}, nil, zap.NewNop())

	c.handleFetchResult(context.Background(), 1, erc20Config(account, fails), erc20FetchResult(1, account, map[common.Address]*big.Int{}, fails))
	stored := storedERC20Balances(t, c, account)
	require.NotContains(t, stored.Balances, fails)
	require.NotContains(t, stored.Answered, fails, "unknown, not zero")
}
