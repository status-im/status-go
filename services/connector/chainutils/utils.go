package chainutils

import (
	"errors"

	"github.com/status-im/status-go/params"
)

type NetworkManagerInterface interface {
	GetActiveNetworks() ([]*params.Network, error)
}

var ErrNoActiveNetworks = errors.New("no active networks available")

// GetSupportedChainIDs retrieves the chain IDs from the provided NetworkManager.
func GetSupportedChainIDs(networkManager NetworkManagerInterface) ([]uint64, error) {
	activeNetworks, err := networkManager.GetActiveNetworks()
	if err != nil {
		return nil, err
	}

	if len(activeNetworks) < 1 {
		return nil, ErrNoActiveNetworks
	}

	chainIDs := make([]uint64, len(activeNetworks))
	for i, network := range activeNetworks {
		chainIDs[i] = network.ChainID
	}

	return chainIDs, nil
}

func GetDefaultChainID(networkManager NetworkManagerInterface) (uint64, error) {
	chainIDs, err := GetSupportedChainIDs(networkManager)
	if err != nil {
		return 0, err
	}

	return chainIDs[0], nil
}
