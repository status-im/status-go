package rpc

import (
	"github.com/status-im/status-go/internal/healthmanager"
)

type EventBlockchainHealthChanged struct {
	FullStatus healthmanager.BlockchainFullStatus
}
