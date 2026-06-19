package wakuv2

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/api/history"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"

	common "github.com/status-im/status-go/pkg/messaging/waku/common"
	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// storenodeAvailableTimeout bounds how long Query waits for a usable store node
// before giving up. It mirrors the timeout historically used by the store-node
// request manager in the protocol package, and guarantees that a caller passing
// a deadline-less context (e.g. the long-lived messenger context) cannot block
// forever when no store node is reachable.
const storenodeAvailableTimeout = 30 * time.Second

// ErrStorenodeNotAvailable is returned by StoreClient.Query when no store node
// becomes available within storenodeAvailableTimeout.
var ErrStorenodeNotAvailable = errors.New("store node is not available")

// StoreClient is the single status-go seam for historic-message retrieval. It
// hides store-node selection behind one method (Query), so callers no longer
// thread a peer through the go-waku quartet
// (WaitForAvailableStoreNode → GetActiveStorenodePeerInfo →
// PerformStorenodeTask → HistoryRetriever.Query).
//
// It is a facade over two collaborators, keeping their responsibilities separate:
//   - selector ("cycle"): which peer, health, failover, pinning.
//   - pager ("retriever"): cursor loop, topic chunking, early-stop, dispatch.
//
// Invariant: a peer is selected once per Query and reused for every page of that
// query; failover happens only at a query boundary (a fresh Query), never
// mid-pagination — a store cursor is bound to the peer that issued it.
//
// Phase 1 wraps the existing go-waku machinery unchanged. Phase 2 will swap the
// collaborators for the Logos Delivery store call without touching callers.
type StoreClient struct {
	cycle              *history.StorenodeCycle
	retriever          *history.HistoryRetriever
	resolvePubsubTopic func(string) string
	logger             *zap.Logger
}

// NewStoreClient builds a StoreClient over an existing storenode cycle (selector)
// and history retriever (pager). resolvePubsubTopic maps an empty pubsub topic to
// the configured default shard topic (see Waku.GetPubsubTopic).
func NewStoreClient(cycle *history.StorenodeCycle, retriever *history.HistoryRetriever, resolvePubsubTopic func(string) string, logger *zap.Logger) *StoreClient {
	return &StoreClient{
		cycle:              cycle,
		retriever:          retriever,
		resolvePubsubTopic: resolvePubsubTopic,
		logger:             logger.Named("store-client"),
	}
}

// Query retrieves historic messages for a single batch (one pubsub topic and its
// content topics over [batch.From, batch.To]) from a single store node, selecting
// the peer itself and paginating until the range is exhausted or
// shouldProcessNextPage stops it early.
//
// pageLimit is the page size of the first request; shouldProcessNextPage (may be
// nil) decides, per page, whether to fetch the next one and with which size;
// processEnvelopes controls whether fetched envelopes are passed to the message
// pipeline synchronously.
func (sc *StoreClient) Query(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	// selector: wait for and pin a store node. The wait is bounded so a
	// deadline-less ctx cannot block indefinitely when no node is reachable.
	waitCtx, cancel := context.WithTimeout(ctx, storenodeAvailableTimeout)
	defer cancel()
	if !sc.cycle.WaitForAvailableStoreNode(waitCtx) {
		return ErrStorenodeNotAvailable
	}
	storenode := sc.cycle.GetActiveStorenodePeerInfo()

	// pager: run every page of this query against the pinned peer.
	// PerformStorenodeTask retries the same peer and only fails over across
	// whole-Query boundaries, never mid-pagination.
	return sc.cycle.PerformStorenodeTask(func() error {
		criteria := store.FilterCriteria{
			TimeStart:     proto.Int64(batch.From.UnixNano()),
			TimeEnd:       proto.Int64(batch.To.UnixNano()),
			ContentFilter: protocol.NewContentFilter(sc.resolvePubsubTopic(batch.PubsubTopic), sc.contentTopics(batch)...),
		}
		return sc.retriever.Query(ctx, criteria, storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
	}, history.WithPeerID(storenode.ID))
}

func (sc *StoreClient) contentTopics(batch types.MailserverBatch) []string {
	contentTopics := make([]string, 0, len(batch.Topics))
	for _, topic := range batch.Topics {
		contentTopics = append(contentTopics, common.BytesToTopic(topic.Bytes()).ContentTopic())
	}
	return contentTopics
}
