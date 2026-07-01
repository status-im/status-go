package messaging

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	adapters "github.com/status-im/status-go/pkg/messaging/adapters"
	types "github.com/status-im/status-go/pkg/messaging/types"
)

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

// SetStorenodes sets the storenodes the StoreClient may query. Called once at
// startup. Storenodes whose addressing info can't be resolved are skipped.
func (a *API) SetStorenodes(nodes []types.StoreNode) {
	addrInfos := make([]peer.AddrInfo, 0, len(nodes))
	for _, node := range nodes {
		info, err := node.PeerInfo()
		if err != nil {
			a.core.logger.Warn("skipping storenode with unresolvable peer info",
				zap.String("id", node.ID), zap.String("name", node.Name), zap.Error(err))
			continue
		}
		addrInfos = append(addrInfos, info)
	}
	if len(addrInfos) == 0 && len(nodes) > 0 {
		a.core.logger.Warn("no usable storenodes after resolving peer info; history queries will fail until reconfigured",
			zap.Int("configured", len(nodes)))
	}
	a.core.stack.Transport.SetStorenodes(addrInfos)
}
