package tokenbalances

import "github.com/ethereum/go-ethereum/common"

// Event types for publisher (used by tokenhistoricalownership service)
type EventBalanceFetchStarted struct {
	ChainID  uint64
	Accounts []common.Address
}

type EventBalanceFetchFinished struct {
	ChainID        uint64
	Account        common.Address
	BalanceChanged bool
}

type EventBalanceFetchError struct {
	ChainID uint64
	Account common.Address
	Error   error
}
