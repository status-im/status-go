package efp

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// FollowingDataProvider defines the interface for providers that can fetch following addresses
type FollowingDataProvider interface {
	ID() string
	IsConnected() bool
	FetchFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]FollowingAddress, error)
}
