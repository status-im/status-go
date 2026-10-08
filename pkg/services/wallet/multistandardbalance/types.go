package multistandardbalance

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
)

type AccountAddress = multistandardfetcher.AccountAddress
type ContractAddress = multistandardfetcher.ContractAddress
type CollectibleID = multistandardfetcher.CollectibleID
type HashableCollectibleID = multistandardfetcher.HashableCollectibleID

type BalancesKey struct {
	Account AccountAddress
	ChainID uint64
}

type State struct {
	AtBlockNumber *big.Int
	AtBlockHash   common.Hash
	FetchedAt     int64
}

// ERC20Balances is the stored ERC20 state of an account on a chain. Balances
// holds the non-zero balances and the last known ones of tokens whose call
// failed; Answered holds every token a fetch answered for. A token answered
// but not in Balances has a zero balance; any other token is unknown.
// Answered is shared between versions while it does not grow: never mutate it.
type ERC20Balances struct {
	Balances map[ContractAddress]*big.Int
	Answered map[ContractAddress]struct{}
}

const NeverFetched = int64(-1)

func defaultState() State {
	return State{
		AtBlockNumber: nil,
		AtBlockHash:   common.Hash{},
		FetchedAt:     NeverFetched,
	}
}
