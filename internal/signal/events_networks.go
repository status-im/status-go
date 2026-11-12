package signal

import (
	"github.com/status-im/status-go/internal/healthmanager"
)

const (
	EventNetworksBlockchainHealthChanged = "networks.blockchainHealthChanged"
)

// BlockchainHealthChangedSignal is triggered when the rpc client for some blockchain goes up/down.
type BlockchainHealthChangedSignal healthmanager.BlockchainFullStatus

func SendNetworksBlockchainHealthChanged(status healthmanager.BlockchainFullStatus) {
	CurrentSession().SendNetworksBlockchainHealthChanged(status)
}

// SendNetworksBlockchainHealthChanged sends in the session s.
func (s Session) SendNetworksBlockchainHealthChanged(status healthmanager.BlockchainFullStatus) {
	s.send(EventNetworksBlockchainHealthChanged, BlockchainHealthChangedSignal(status))
}
