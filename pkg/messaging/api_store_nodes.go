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

func (a *API) WaitForAvailableStoreNode(ctx context.Context) bool {
	return a.core.stack.Transport.WaitForAvailableStoreNode(ctx)
}

func (a *API) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return a.core.stack.Transport.PerformStorenodeTask(fn, opts...)
}

// StoreQuery executes an explicit historical message query against a store
// node, decoupled from chat filters. The caller resolves filters into the
// criteria carried by req (content topics + pubsub topic + time range); peer
// selection is delegated to the underlying store ("at own risk").
func (a *API) StoreQuery(ctx context.Context, req types.StoreQueryRequest) error {
	batch := types.StoreNodeBatch{
		From:        req.From,
		To:          req.To,
		PubsubTopic: req.PubsubTopic,
		Topics:      req.ContentTopics,
	}
	return a.core.stack.Transport.StoreQuery(ctx, *adapters.ToWakuBatch(&batch), req.PageSize, req.ShouldProcessNextPage, req.ProcessEnvelopes)
}

func (a *API) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	a.core.stack.Transport.SetStorenodeConfigProvider(c)
}
