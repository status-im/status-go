package transferdetector

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

type EventTransferDetectionStarted struct {
	ChainID   uint64
	Accounts  []common.Address
	FromBlock uint64
	ToBlock   uint64
}

type EventTransferDetectionFinished struct {
	ChainID   uint64
	Accounts  []common.Address
	FromBlock uint64
	ToBlock   uint64
	Events    []eventlog.Event
}

type EventTransferDetectionError struct {
	ChainID   uint64
	Accounts  []common.Address
	FromBlock uint64
	ToBlock   uint64
	Error     error
}
