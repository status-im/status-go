package waku

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/libp2p/go-libp2p/core/peer"

	common "github.com/status-im/status-go/pkg/messaging/waku/common"
	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// maxStoreQueryAttempts bounds how many storenodes Query tries before giving up.
const maxStoreQueryAttempts = 3

// ErrNoStorenodesReachable is returned when no configured storenode could serve
// the query (none configured, or all attempts failed).
var ErrNoStorenodesReachable = errors.New("no store node could serve the query")

// StoreClient is the single status-go seam for historic-message retrieval: one
// Query method, no peer argument. It selects a storenode on demand — no health
// checks, no background cycle, no pinning — and fails over to another node if a
// query fails. Phase 2 will swap the pager's wire call for the Logos Delivery
// store FFI without touching callers.
type StoreClient struct {
	selector           *storeSelector
	pager              *storePager
	resolvePubsubTopic func(string) string
	logger             *zap.Logger
}

func NewStoreClient(selector *storeSelector, pager *storePager, resolvePubsubTopic func(string) string, logger *zap.Logger) *StoreClient {
	return &StoreClient{
		selector:           selector,
		pager:              pager,
		resolvePubsubTopic: resolvePubsubTopic,
		logger:             logger.Named("store-client"),
	}
}

// SetStorenodes sets the storenodes Query may use. Called once at startup.
func (sc *StoreClient) SetStorenodes(nodes []peer.AddrInfo) {
	sc.selector.setStorenodes(nodes)
}

// nextStorenode returns a currently eligible storenode for APIs that need one
// concrete peer, such as targeted hash retrieval.
func (sc *StoreClient) nextStorenode() peer.AddrInfo {
	candidates := sc.selector.candidates()
	if len(candidates) == 0 {
		return peer.AddrInfo{}
	}
	return candidates[0]
}

// Query retrieves historic messages for a single batch (one pubsub topic and its
// content topics over [batch.From, batch.To]). It tries storenodes in turn until
// one serves the whole query; on a mid-query failure it restarts on the next node
// (a store cursor is bound to the node that issued it). pageLimit is the first
// page size; shouldProcessNextPage (may be nil) is the per-page early-stop;
// processEnvelopes controls synchronous handling of fetched envelopes.
func (sc *StoreClient) Query(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	candidates := sc.selector.candidates()
	if len(candidates) == 0 {
		return ErrNoStorenodesReachable
	}

	pubsubTopic := sc.resolvePubsubTopic(batch.PubsubTopic)
	contentTopics := sc.contentTopics(batch)

	var lastErr error
	for i, node := range candidates {
		if i >= maxStoreQueryAttempts {
			break
		}
		err := sc.pager.run(ctx, node, pubsubTopic, contentTopics, batch.From, batch.To, pageLimit, shouldProcessNextPage, processEnvelopes)
		if err == nil {
			sc.selector.markSuccess(node.ID)
			return nil
		}
		if ctx.Err() != nil {
			return err // caller cancelled — don't keep trying
		}
		sc.selector.markFailure(node.ID)
		lastErr = err
		sc.logger.Debug("store query failed, trying next storenode",
			zap.Stringer("peerID", node.ID), zap.Error(err))
	}

	if lastErr == nil {
		lastErr = ErrNoStorenodesReachable
	}
	return lastErr
}

func (sc *StoreClient) contentTopics(batch types.MailserverBatch) []string {
	contentTopics := make([]string, 0, len(batch.Topics))
	for _, topic := range batch.Topics {
		contentTopics = append(contentTopics, common.BytesToTopic(topic.Bytes()).ContentTopic())
	}
	return contentTopics
}
