package tokenbalances

import (
	"context"
	"maps"
	"math/big"

	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

type StorageMultistandardBalance struct {
	multistandardbalanceStorage multistandardbalance.Storage
}

func NewStorageMultistandardBalance(multistandardbalanceStorage multistandardbalance.Storage) *StorageMultistandardBalance {
	return &StorageMultistandardBalance{multistandardbalanceStorage: multistandardbalanceStorage}
}

func (s *StorageMultistandardBalance) GetBalances(ctx context.Context, tokens []*tokentypes.Token, accountAddresses []AccountAddress) (
	map[uint64]map[AccountAddress]map[ContractAddress]*big.Int, error) {
	ret := make(map[uint64]map[AccountAddress]map[ContractAddress]*big.Int)

	tokensPerChain := make(map[uint64][]*tokentypes.Token)
	for _, token := range tokens {
		tokensPerChain[token.ChainID] = append(tokensPerChain[token.ChainID], token)
	}

	for chainID, chainTokens := range tokensPerChain {
		ret[chainID] = make(map[AccountAddress]map[ContractAddress]*big.Int)
		for _, account := range accountAddresses {
			ret[chainID][account] = make(map[ContractAddress]*big.Int)
			erc20balances, _, err := s.multistandardbalanceStorage.GetERC20Balances(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
			if err != nil {
				return nil, err
			}
			needsNative := false
			for _, token := range chainTokens {
				if token.IsNative() {
					needsNative = true
				}
				if _, exists := erc20balances[token.Address]; exists {
					ret[chainID][account][token.Address] = erc20balances[token.Address]
				}
			}
			if needsNative {
				nativeBalance, _, err := s.multistandardbalanceStorage.GetNativeBalance(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
				if err != nil {
					return nil, err
				}
				ret[chainID][account][NativeTokenAddress] = nativeBalance
			}
		}
	}
	return ret, nil
}

func (s *StorageMultistandardBalance) GetAllBalancesForAccountAndChain(ctx context.Context, accountAddress AccountAddress, chainID uint64) (map[ContractAddress]*big.Int, State, error) {
	balances := make(map[ContractAddress]*big.Int)

	// Add all ERC20 balances
	erc20balances, state, err := s.multistandardbalanceStorage.GetERC20Balances(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: accountAddress})
	if err != nil {
		return nil, state, err
	}
	maps.Copy(balances, erc20balances)

	// Add the native token
	nativeBalance, state, err := s.multistandardbalanceStorage.GetNativeBalance(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: accountAddress})
	if err != nil {
		return nil, state, err
	}
	balances[NativeTokenAddress] = nativeBalance

	return balances, state, nil
}
