package tokenevent

import (
	"github.com/ethereum/go-ethereum/common"
)

// TokenDiscoveryRequestEvent is emitted when a component needs token metadata.
// The TokenManager will asynchronously discover the token.
type TokenDiscoveryRequestEvent struct {
	ChainID uint64
	Address common.Address
}
