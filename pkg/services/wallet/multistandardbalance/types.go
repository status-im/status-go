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

const NeverFetched = int64(-1)

func defaultState() State {
	return State{
		AtBlockNumber: nil,
		AtBlockHash:   common.Hash{},
		FetchedAt:     NeverFetched,
	}
}
