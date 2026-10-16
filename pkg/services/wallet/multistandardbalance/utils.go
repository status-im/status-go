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

// countMissing counts the stored tokens a read did not answer. A batched
// balance fetch (multicall with requireSuccess=false) drops the sub-calls that
// failed; the storage keeps their last known balances.
func countMissing[T comparable](stored map[T]*big.Int, read map[T]*big.Int) int {
	missing := 0
	for token := range stored {
		if _, answered := read[token]; !answered {
			missing++
		}
	}
	return missing
}
