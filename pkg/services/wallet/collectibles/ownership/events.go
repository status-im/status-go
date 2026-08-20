package ownership

import (
	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

// LoaderID identifies the Loader that published the event. A (ChainID, Account)
// pair doesn't identify a load: a loader restart replaces the loader for the
// pair while the replaced one is still unwinding, so events from both can reach
// the same subscriber. Waiters interested in one specific load must match on it.
type EventOwnedCollectiblesLoadStarted struct {
	ChainID  walletCommon.ChainID
	Account  common.Address
	LoaderID uint64
}

type EventOwnedCollectiblesLoadPartial struct {
	ChainID          walletCommon.ChainID
	Account          common.Address
	LoaderID         uint64
	Added            []thirdparty.CollectibleUniqueID
	PartialOwnership []thirdparty.CollectibleIDBalance
}

type EventOwnedCollectiblesLoadFinished struct {
	ChainID      walletCommon.ChainID
	Account      common.Address
	LoaderID     uint64
	Added        []thirdparty.CollectibleUniqueID
	Updated      []thirdparty.CollectibleUniqueID
	Removed      []thirdparty.CollectibleUniqueID
	NewOwnership []thirdparty.CollectibleIDBalance
}

type EventOwnedCollectiblesLoadError struct {
	ChainID  walletCommon.ChainID
	Account  common.Address
	LoaderID uint64
	Error    error
}

// Published when a load ends because it was cancelled (loader stopped or
// restarted). Internal: lets waiters on a running load unblock; not translated
// into a client event.
type EventOwnedCollectiblesLoadCancelled struct {
	ChainID  walletCommon.ChainID
	Account  common.Address
	LoaderID uint64
}
