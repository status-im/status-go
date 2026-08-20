package tokenbalances

//go:generate go tool mockgen -package=mock_tokenbalances -source=fetcher.go -destination=mock/fetcher.go

import (
	"context"
	"fmt"
	"maps"
	"math/big"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/internal/logutils"
)

type MultiStandardBalanceFetcher interface {
	FetchBalances(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error)
}

type FetcherIface interface {
	Fetch(ctx context.Context, chainID uint64, tokenAddresses []ContractAddress, accountAddresses []AccountAddress) (map[AccountAddress]map[ContractAddress]*big.Int, error)
	FetchSingle(ctx context.Context, chainID uint64, tokenAddress ContractAddress, accountAddress AccountAddress) (*big.Int, error)
}

type Fetcher struct {
	multiStandardBalanceFetcher MultiStandardBalanceFetcher
}

func NewFetcher(multiStandardBalanceFetcher MultiStandardBalanceFetcher) *Fetcher {
	return &Fetcher{
		multiStandardBalanceFetcher: multiStandardBalanceFetcher,
	}
}

func (f *Fetcher) Fetch(ctx context.Context, chainID uint64, tokenAddresses []ContractAddress, accountAddresses []AccountAddress) (map[AccountAddress]map[ContractAddress]*big.Int, error) {
	config := multistandardfetcher.FetchConfig{
		Native: make([]AccountAddress, 0, len(accountAddresses)),
		ERC20:  make(map[AccountAddress][]ContractAddress),
	}

	for _, tokenAddress := range tokenAddresses {
		if tokenAddress == NativeTokenAddress {
			// Native token represented by zero address
			config.Native = append(config.Native, accountAddresses...)
		} else {
			for _, accountAddress := range accountAddresses {
				config.ERC20[accountAddress] = append(config.ERC20[accountAddress], tokenAddress)
			}
		}
	}

	resultsCh, err := f.multiStandardBalanceFetcher.FetchBalances(ctx, chainID, config)
	if err != nil {
		return nil, err
	}

	ret := make(map[AccountAddress]map[ContractAddress]*big.Int)

	for result := range resultsCh {
		switch result.ResultType {
		case multistandardfetcher.ResultTypeNative:
			result, ok := result.Result.(multistandardfetcher.NativeResult)
			if !ok {
				return nil, fmt.Errorf("failed to parse native result")
			}
			if result.Err != nil {
				return nil, result.Err
			}
			if ret[result.Account] == nil {
				ret[result.Account] = make(map[ContractAddress]*big.Int)
			}
			ret[result.Account][NativeTokenAddress] = result.Result
		case multistandardfetcher.ResultTypeERC20:
			result, ok := result.Result.(multistandardfetcher.ERC20Result)
			if !ok {
				return nil, fmt.Errorf("failed to parse ERC20 result")
			}
			if result.Err != nil {
				return nil, result.Err
			}
			if ret[result.Account] == nil {
				ret[result.Account] = make(map[ContractAddress]*big.Int)
			}
			maps.Copy(ret[result.Account], result.Results)
		}
	}

	return ret, nil
}

func (f *Fetcher) FetchSingle(ctx context.Context, chainID uint64, tokenAddress ContractAddress, accountAddress AccountAddress) (*big.Int, error) {
	balances, err := f.Fetch(ctx, chainID, []ContractAddress{tokenAddress}, []AccountAddress{accountAddress})
	if err != nil {
		return nil, err
	}
	if balancesPerTokenAddress, ok := balances[accountAddress]; ok {
		if balance, ok := balancesPerTokenAddress[tokenAddress]; ok {
			return balance, nil
		}
	}
	return nil, fmt.Errorf("no balance found for chain %d, token %s, account %s", chainID, tokenAddress, logutils.TruncateWithDot(accountAddress.String()))
}
