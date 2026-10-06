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

// mergeERC20Balances builds the ERC20 balances to store from a fetch result,
// reusing the fetched map. Zero balances are not stored: a token missing from
// the map reads as zero. A batched fetch (multicall with requireSuccess=false)
// drops the sub-calls that failed; such a token keeps its stored entry, or is
// stored as nil (unknown) when there is none, rather than reading as a hard zero
// until a later fetch answers it. So does a stored token the fetch did not ask
// for. Returns the map and how many calls failed.
func mergeERC20Balances(requested []ContractAddress, previous, fetched map[ContractAddress]*big.Int, failed []ContractAddress) (map[ContractAddress]*big.Int, int) {
	if fetched == nil {
		fetched = make(map[ContractAddress]*big.Int)
	}
	for token, balance := range fetched {
		if balance == nil || balance.Sign() == 0 {
			delete(fetched, token)
		}
	}
	for _, token := range failed {
		fetched[token] = previous[token]
	}
	if len(previous) > 0 {
		notRequested := make(map[ContractAddress]struct{}, len(previous))
		for token := range previous {
			notRequested[token] = struct{}{}
		}
		for _, token := range requested {
			delete(notRequested, token)
		}
		for token := range notRequested {
			fetched[token] = previous[token]
		}
	}
	return fetched, len(failed)
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
		if v1 == nil || v2 == nil {
			if v1 != v2 {
				return false
			}
			continue
		}
		if v1.Cmp(v2) != 0 {
			return false
		}
	}
	return true
}
