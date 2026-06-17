package wakuv2

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p/core/peer"

	common "github.com/status-im/status-go/pkg/messaging/waku/common"
	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// maxStoreQueryAttempts bounds how many storenodes an operation tries before giving up.
const maxStoreQueryAttempts = 3

// ErrNoStorenodesReachable is returned when no configured storenode could serve
// the query (none configured, or all attempts failed).
var ErrNoStorenodesReachable = errors.New("no store node could serve the query")

// StoreClient is the single status-go seam for store access: explicit history
// retrieval (Query) and background missing-message reconciliation
// (FetchMissingMessages). It selects a storenode on demand — no health checks, no
// background cycle, no pinning — pinning one node for the duration of a single
// operation and failing over to another only between operations. Phase 2 will swap
// the wire call for the Logos Delivery store FFI without touching callers.
type StoreClient struct {
	selector           *storeSelector
	requestor          storeRequestor
	pager              *storePager
	resolvePubsubTopic func(string) string
	logger             *zap.Logger
}

func NewStoreClient(selector *storeSelector, requestor storeRequestor, processor envelopeProcessor, resolvePubsubTopic func(string) string, logger *zap.Logger) *StoreClient {
	return &StoreClient{
		selector:           selector,
		requestor:          requestor,
		pager:              newStorePager(requestor, processor, logger),
		resolvePubsubTopic: resolvePubsubTopic,
		logger:             logger.Named("store-client"),
	}
}

// SetStorenodes sets the storenodes operations may use. Called once at startup.
func (sc *StoreClient) SetStorenodes(nodes []peer.AddrInfo) {
	sc.selector.setStorenodes(nodes)
}

// HasStorenodes reports whether any storenode is configured (used to gate sync).
func (sc *StoreClient) HasStorenodes() bool {
	return sc.selector.hasStorenodes()
}

// withStorenode runs fn against configured storenodes in turn, marking each
// success/failure on the selector and stopping at the first success. fn receives a
// single storenode and must run its whole operation against it — a store cursor is
// bound to the node that issued it, so a mid-operation failure restarts fn on the
// next node. Returns ErrNoStorenodesReachable if none is configured or every
// bounded attempt failed.
func (sc *StoreClient) withStorenode(ctx context.Context, fn func(ctx context.Context, node peer.AddrInfo) error) error {
	candidates := sc.selector.candidates()
	if len(candidates) == 0 {
		return ErrNoStorenodesReachable
	}

	var lastErr error
	for i, node := range candidates {
		if i >= maxStoreQueryAttempts {
			break
		}
		err := fn(ctx, node)
		if err == nil {
			sc.selector.markSuccess(node.ID)
			return nil
		}
		if ctx.Err() != nil {
			return err // caller cancelled — don't keep trying
		}
		sc.selector.markFailure(node.ID)
		lastErr = err
		sc.logger.Debug("store operation failed, trying next storenode",
			zap.Stringer("peerID", node.ID), zap.Error(err))
	}

	if lastErr == nil {
		lastErr = ErrNoStorenodesReachable
	}
	return lastErr
}

// Query retrieves historic messages for a single batch (one pubsub topic and its
// content topics over [batch.From, batch.To]). It pins one storenode for the whole
// query and fails over to another only if that node fails. pageLimit is the first
// page size; shouldProcessNextPage (may be nil) is the per-page early-stop;
// processEnvelopes controls synchronous handling of fetched envelopes.
func (sc *StoreClient) Query(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	pubsubTopic := sc.resolvePubsubTopic(batch.PubsubTopic)
	contentTopics := sc.contentTopics(batch)
	return sc.withStorenode(ctx, func(ctx context.Context, node peer.AddrInfo) error {
		return sc.pager.run(ctx, node, pubsubTopic, contentTopics, batch.From, batch.To, pageLimit, shouldProcessNextPage, processEnvelopes)
	})
}

func (sc *StoreClient) contentTopics(batch types.MailserverBatch) []string {
	contentTopics := make([]string, 0, len(batch.Topics))
	for _, topic := range batch.Topics {
		contentTopics = append(contentTopics, common.BytesToTopic(topic.Bytes()).ContentTopic())
	}
	return contentTopics
}
