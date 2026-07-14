package messaging

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/api/history"

	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// These methods ideally shouldn't be exposed as they reveal implementation details
// of how the messaging module retrieves historic messages. This is a transitional
// approach pending Waku SDK integration.

func (a *API) GetActiveStorenode() peer.AddrInfo {
	return a.core.stack.Transport.GetActiveStorenode()
}

func (a *API) DisconnectActiveStorenode(ctx context.Context, backoffReason time.Duration, shouldCycle bool) {
	a.core.stack.Transport.DisconnectActiveStorenode(ctx, backoffReason, shouldCycle)
}

func (a *API) OnStorenodeChanged() <-chan peer.ID {
	return a.core.stack.Transport.OnStorenodeChanged()
}

func (a *API) OnStorenodeNotWorking() <-chan struct{} {
	return a.core.stack.Transport.OnStorenodeNotWorking()
}

func (a *API) OnStorenodeAvailable() <-chan peer.ID {
	return a.core.stack.Transport.OnStorenodeAvailable()
}

func (a *API) WaitForAvailableStoreNode(ctx context.Context) bool {
	return a.core.stack.Transport.WaitForAvailableStoreNode(ctx)
}

func (a *API) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return a.core.stack.Transport.PerformStorenodeTask(fn, opts...)
}

func (a *API) ProcessMailserverBatch(
	ctx context.Context,
	batch types.StoreNodeBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.core.stack.Transport.ProcessMailserverBatch(ctx, *adapters.ToWakuBatch(&batch), storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
}

// ProcessMailserverBatchHashFirst runs a hash-first store-node backfill of the batch,
// fetching full bodies only for hashes not already held locally (issue #21470-hf). It
// returns per-batch stats for the caller's once-per-backfill log.
func (a *API) ProcessMailserverBatchHashFirst(
	ctx context.Context,
	batch types.StoreNodeBatch,
	storenode peer.AddrInfo,
	processEnvelopes bool,
) (wakutypes.HashFirstStats, error) {
	return a.core.stack.Transport.ProcessMailserverBatchHashFirst(ctx, *adapters.ToWakuBatch(&batch), storenode, processEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.core.stack.Transport.SetStorenodeConfigProvider(c)
}

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters types.ChatFilters) {
	a.core.stack.Transport.SetCriteriaForMissingMessageVerification(peerInfo, adapters.ToTransportFilters(filters))
}
