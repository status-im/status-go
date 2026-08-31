package tokenbalances

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	wCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

type fakeMultiStandardBalanceFetcher struct {
	fetchBalances func(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error)
}

func (f *fakeMultiStandardBalanceFetcher) FetchBalances(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
	return f.fetchBalances(ctx, chainID, config)
}

func resultsChannel(results ...multistandardfetcher.FetchResult) <-chan multistandardfetcher.FetchResult {
	ch := make(chan multistandardfetcher.FetchResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

func nativeResult(account AccountAddress, balance *big.Int) multistandardfetcher.FetchResult {
	return multistandardfetcher.FetchResult{
		ResultType: multistandardfetcher.ResultTypeNative,
		Result: multistandardfetcher.NativeResult{
			Account: account,
			Result:  balance,
		},
	}
}

// ETH on ZKsync Era is exposed by token lists at the 0x…800a system-contract alias, which is
// not a standard ERC20. FetchSingle must resolve such aliases through the native path and
// still key the result under the requested alias address.
func TestFetchSingleNativeTokenAlias(t *testing.T) {
	account := AccountAddress{0x11}
	balance := big.NewInt(42)

	fake := &fakeMultiStandardBalanceFetcher{
		fetchBalances: func(_ context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
			require.Equal(t, wCommon.ZkSyncMainnet, chainID)
			// The alias must be fetched natively, never as an ERC20 contract call.
			require.Equal(t, []AccountAddress{account}, config.Native)
			require.Empty(t, config.ERC20)
			return resultsChannel(nativeResult(account, balance)), nil
		},
	}

	fetcher := NewFetcher(fake)
	got, err := fetcher.FetchSingle(context.Background(), wCommon.ZkSyncMainnet, wCommon.ZkSyncETHTokenAddress(), account)
	require.NoError(t, err)
	require.Equal(t, balance, got)
}

// Requesting the canonical zero address and an alias together must produce a single native
// fetch, with the balance mirrored under both requested addresses.
func TestFetchNativeTokenAliasDeduplicated(t *testing.T) {
	account := AccountAddress{0x11}
	balance := big.NewInt(7)

	fake := &fakeMultiStandardBalanceFetcher{
		fetchBalances: func(_ context.Context, _ uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
			require.Equal(t, []AccountAddress{account}, config.Native) // accounts added once, not per address
			require.Empty(t, config.ERC20)
			return resultsChannel(nativeResult(account, balance)), nil
		},
	}

	fetcher := NewFetcher(fake)
	balances, err := fetcher.Fetch(context.Background(), wCommon.ZkSyncMainnet,
		[]ContractAddress{NativeTokenAddress, wCommon.ZkSyncETHTokenAddress()}, []AccountAddress{account})
	require.NoError(t, err)
	require.Equal(t, balance, balances[account][NativeTokenAddress])
	require.Equal(t, balance, balances[account][wCommon.ZkSyncETHTokenAddress()])
}

// On chains without a registered alias, a non-zero token address keeps going down the ERC20 path.
func TestFetchSingleERC20Unaffected(t *testing.T) {
	account := AccountAddress{0x11}
	tokenAddress := ContractAddress{0x22}
	balance := big.NewInt(1000)

	fake := &fakeMultiStandardBalanceFetcher{
		fetchBalances: func(_ context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
			require.Equal(t, wCommon.EthereumMainnet, chainID)
			require.Empty(t, config.Native)
			require.Equal(t, []ContractAddress{tokenAddress}, config.ERC20[account])
			return resultsChannel(multistandardfetcher.FetchResult{
				ResultType: multistandardfetcher.ResultTypeERC20,
				Result: multistandardfetcher.ERC20Result{
					Account: account,
					Results: map[ContractAddress]*big.Int{tokenAddress: balance},
				},
			}), nil
		},
	}

	fetcher := NewFetcher(fake)
	got, err := fetcher.FetchSingle(context.Background(), wCommon.EthereumMainnet, tokenAddress, account)
	require.NoError(t, err)
	require.Equal(t, balance, got)
}
