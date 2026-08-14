package ownership

import (
	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type EventOwnedCollectiblesLoadStarted struct {
	ChainID walletCommon.ChainID
	Account common.Address
}

type EventOwnedCollectiblesLoadPartial struct {
	ChainID          walletCommon.ChainID
	Account          common.Address
	Added            []thirdparty.CollectibleUniqueID
	PartialOwnership []thirdparty.CollectibleIDBalance
}

type EventOwnedCollectiblesLoadFinished struct {
	ChainID      walletCommon.ChainID
	Account      common.Address
	Added        []thirdparty.CollectibleUniqueID
	Updated      []thirdparty.CollectibleUniqueID
	Removed      []thirdparty.CollectibleUniqueID
	NewOwnership []thirdparty.CollectibleIDBalance
}

type EventOwnedCollectiblesLoadError struct {
	ChainID walletCommon.ChainID
	Account common.Address
	Error   error
}

// Published when a load ends because it was cancelled (loader stopped or
// restarted). Internal: lets waiters on a running load unblock; not translated
// into a client event.
type EventOwnedCollectiblesLoadCancelled struct {
	ChainID walletCommon.ChainID
	Account common.Address
}
