package efp

//go:generate go tool mockgen -package=mock_efp -source=types.go -destination=mock/types.go

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// FollowingDataProvider defines the interface for providers that can fetch following addresses
type FollowingDataProvider interface {
	ID() string
	IsConnected() bool
	FetchFollowingAddresses(ctx context.Context, userAddress common.Address) ([]FollowingAddress, error)
}

