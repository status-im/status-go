package common

import (
	"context"

	"github.com/status-im/status-go/params"
)

// ShouldCancel returns true if the context has been cancelled and task should be aborted
func ShouldCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

func NetworksToChainIDs(networks []*params.Network) []uint64 {
	chainIDs := make([]uint64, 0)
	for _, network := range networks {
		chainIDs = append(chainIDs, network.ChainID)
	}

	return chainIDs
}
