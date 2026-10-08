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

// mergeERC20Balances builds the stored ERC20 state from the previous one and a
// fetch that asked for the asked tokens. fetched holds the non-zero balances it
// answered (reused); failed the tokens whose call failed. A token the fetch
// answered and left out of fetched is a zero. Previous balances of failed and
// of not asked tokens are kept.
func mergeERC20Balances(previous ERC20Balances, asked []ContractAddress, fetched map[ContractAddress]*big.Int, failed []ContractAddress) ERC20Balances {
	if fetched == nil {
		fetched = make(map[ContractAddress]*big.Int)
	}
	var failedSet map[ContractAddress]struct{}
	if len(failed) > 0 {
		failedSet = make(map[ContractAddress]struct{}, len(failed))
		for _, token := range failed {
			failedSet[token] = struct{}{}
		}
	}
	answeredNow := func(token ContractAddress) bool {
		_, isFailed := failedSet[token]
		return !isFailed
	}

	if len(previous.Balances) > 0 {
		kept := make(map[ContractAddress]struct{}, len(previous.Balances))
		for token := range previous.Balances {
			if _, ok := fetched[token]; !ok {
				kept[token] = struct{}{}
			}
		}
		for _, token := range asked {
			if answeredNow(token) {
				delete(kept, token)
			}
		}
		for token := range kept {
			fetched[token] = previous.Balances[token]
		}
	}

	answered := previous.Answered
	grown := false
	for _, token := range asked {
		if !answeredNow(token) {
			continue
		}
		if _, ok := answered[token]; ok {
			continue
		}
		if !grown {
			answered = make(map[ContractAddress]struct{}, len(previous.Answered)+len(asked))
			for t := range previous.Answered {
				answered[t] = struct{}{}
			}
			grown = true
		}
		answered[token] = struct{}{}
	}

	return ERC20Balances{Balances: fetched, Answered: answered}
}

// isERC20BalancesEqual relies on Answered only ever growing.
func isERC20BalancesEqual(b1 ERC20Balances, b2 ERC20Balances) bool {
	return len(b1.Answered) == len(b2.Answered) && isBigIntMapEqual(b1.Balances, b2.Balances)
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
