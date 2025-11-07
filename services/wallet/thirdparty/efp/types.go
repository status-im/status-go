package efp

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

//go:generate go tool mockgen -package=mock_efp -source=types.go -destination=mock/mock_efp.go

// FollowingDataProvider defines the interface for providers that can fetch following addresses
type FollowingDataProvider interface {
	ID() string
	IsConnected() bool
	FetchFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]FollowingAddress, error)
	FetchFollowingStats(ctx context.Context, userAddress common.Address) (int, error)
}
