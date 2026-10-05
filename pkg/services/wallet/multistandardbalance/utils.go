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

// keepLastKnownBalances carries the previously stored balance of every token the
// new fetch did not answer. A batched balance fetch (multicall with
// requireSuccess=false) drops the sub-calls that failed, so a token can be
// missing from the result although it is still in the token list. Storing the
// result as-is would make the reader report that token as a hard zero (no
// error) until a later fetch answers it again. Returns the merged map and how
// many tokens were carried over.
func keepLastKnownBalances[T comparable](previous map[T]*big.Int, fetched map[T]*big.Int) (map[T]*big.Int, int) {
	if fetched == nil {
		fetched = make(map[T]*big.Int)
	}
	kept := 0
	for token, balance := range previous {
		if _, answered := fetched[token]; answered {
			continue
		}
		fetched[token] = balance
		kept++
	}
	return fetched, kept
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
