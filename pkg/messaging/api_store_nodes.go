package messaging

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/api/history"

	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
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

// Query retrieves historic messages for a single batch (one pubsub topic and its
// content topics over [batch.From, batch.To]). The store node is selected
// internally — callers no longer pass a peer. shouldProcessNextPage (may be nil)
// is the per-page early-stop; processEnvelopes controls synchronous handling of
// fetched envelopes.
func (a *API) Query(
	ctx context.Context,
	batch types.StoreNodeBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.core.stack.Transport.Query(ctx, *adapters.ToWakuBatch(&batch), pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.core.stack.Transport.SetStorenodeConfigProvider(c)
}

func (a *API) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters types.ChatFilters) {
	a.core.stack.Transport.SetCriteriaForMissingMessageVerification(peerInfo, adapters.ToTransportFilters(filters))
}
