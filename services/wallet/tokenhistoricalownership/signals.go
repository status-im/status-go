package tokenhistoricalownership

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/signal"
)

const (
	SignalOwnershipChanged = signal.SignalType("wallet.token-historical-ownership.changed")
)

type SignalOwnershipChangedPayload struct {
	ChainID uint64         `json:"chainId"`
	Account common.Address `json:"account"`
}

// Event types for publisher
type EventOwnershipChanged struct {
	ChainID uint64
	Account common.Address
}
