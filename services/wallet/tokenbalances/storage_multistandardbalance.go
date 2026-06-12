package tokenbalances

import (
	"context"
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
			erc20balances, erc20State, err := s.multistandardbalanceStorage.GetERC20Balances(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
			if err != nil {
				return nil, err
			}
			needsNative := false
			for _, token := range chainTokens {
				if token.IsNative() {
					needsNative = true
				}
				if balance, exists := erc20balances[token.Address]; exists {
					ret[chainID][account][token.Address] = balance
				} else if erc20State.FetchedAt != multistandardbalance.NeverFetched {
					ret[chainID][account][token.Address] = big.NewInt(0)
				}
			}
			if needsNative {
				nativeBalance, nativeState, err := s.multistandardbalanceStorage.GetNativeBalance(ctx, multistandardbalance.BalancesKey{ChainID: chainID, Account: account})
				if err != nil {
					return nil, err
				}
				if nativeState.FetchedAt != multistandardbalance.NeverFetched {
					ret[chainID][account][NativeTokenAddress] = nativeBalance
				}
			}
		}
	}
	return ret, nil
}
