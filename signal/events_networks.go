package signal

import (
	"github.com/status-im/status-go/healthmanager"
)

const (
	EventNetworksBlockchainHealthChanged = "networks.blockchainHealthChanged"
)

// BlockchainHealthChangedSignal is triggered when the rpc client for some blockchain goes up/down.
type BlockchainHealthChangedSignal healthmanager.BlockchainFullStatus

func SendNetworksBlockchainHealthChanged(status healthmanager.BlockchainFullStatus) {
	send(EventNetworksBlockchainHealthChanged, BlockchainHealthChangedSignal(status))
}
