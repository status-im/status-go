package tokenbalances

import (
	"context"
	"math/big"

	"github.com/status-im/status-go/services/wallet/multistandardbalance"
)

type StorageMultistandardBalance struct {
	multistandardbalanceStorage multistandardbalance.Storage
}

func NewStorageMultistandardBalance(multistandardbalanceStorage multistandardbalance.Storage) *StorageMultistandardBalance {
	return &StorageMultistandardBalance{multistandardbalanceStorage: multistandardbalanceStorage}
}

func (s *StorageMultistandardBalance) GetBalances(ctx context.Context, chainID uint64, tokenAddresses []ContractAddress, accountAddresses []AccountAddress) (map[AccountAddress]map[ContractAddress]*big.Int, error) {
	ret := make(map[AccountAddress]map[ContractAddress]*big.Int)
	for _, account := range accountAddresses {
		ret[account] = make(map[ContractAddress]*big.Int)
		erc20balances, _, err := s.multistandardbalanceStorage.GetERC20Balances(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
		if err != nil {
			return nil, err
		}
		needsNative := false
		for _, tokenAddress := range tokenAddresses {
			if tokenAddress == NativeTokenAddress {
				needsNative = true
			}
			if _, exists := erc20balances[tokenAddress]; exists {
				ret[account][tokenAddress] = erc20balances[tokenAddress]
			}
		}
		if needsNative {
			nativeBalance, _, err := s.multistandardbalanceStorage.GetNativeBalance(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
			if err != nil {
				return nil, err
			}
			ret[account][NativeTokenAddress] = nativeBalance
		}
	}
	return ret, nil
}
