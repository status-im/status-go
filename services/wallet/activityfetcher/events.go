package activityfetcher

import (
	"github.com/ethereum/go-ethereum/common"
)

// EventERC20ActivityFetched is emitted when ERC20 token transfers are fetched from activity sources.
// This event notifies subscribers that ERC20 activity has been discovered, allowing them to take
// appropriate actions such as fetching token metadata.
type EventERC20ActivityFetched struct {
	ChainID uint64         // The chain ID where the activity was found
	Address common.Address // The ERC20 token contract address
}
