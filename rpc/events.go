package rpc

import (
	"github.com/status-im/status-go/healthmanager"
)

type EventBlockchainHealthChanged struct {
	FullStatus healthmanager.BlockchainFullStatus
}
