package multistandardbalance

import (
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

func deleteAccountsNotInList[T any](m map[BalancesKey]T, accounts []common.Address) {
	for key := range m {
		if !slices.Contains(accounts, key.Account) {
			delete(m, key)
		}
	}
}

func deleteChainsNotInList[T any](m map[BalancesKey]T, chains []uint64) {
	for key := range m {
		if !slices.Contains(chains, key.ChainID) {
			delete(m, key)
		}
	}
}

func isBigIntMapEqual[T comparable](m1 map[T]*big.Int, m2 map[T]*big.Int) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok {
			return false
		}
		if v1.Cmp(v2) != 0 {
			return false
		}
	}
	return true
}
