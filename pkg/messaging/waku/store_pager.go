package waku

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	"github.com/status-im/status-go/internal/panics"
)

const (
	// maxContentTopicsPerRequest is the store protocol's per-request content-topic limit.
	maxContentTopicsPerRequest = 10
	// maxStoreRequestWindow caps a single request's time range; larger ranges are
	// walked one window at a time, newest first.
	maxStoreRequestWindow = 24 * time.Hour
	// maxConcurrentStoreRequests bounds in-flight store requests across the whole
	// pager (shared by every concurrent Query, not per-Query).
	maxConcurrentStoreRequests = 3
	storeRequestTimeout        = 30 * time.Second
)

// envelopeProcessor consumes messages fetched from a storenode.
type envelopeProcessor interface {
	OnEnvelope(env *protocol.Envelope, processEnvelopes bool) error
}

// storePager reads a time range of historic messages from a single storenode. It
// splits content topics into store-protocol-sized requests, walks the range in
// <=24h windows, follows pagination cursors, and stops early when the caller's
// shouldProcessNextPage says so. The storenode is fixed for the whole run, so
// cursors stay valid.
type storePager struct {
	requestor storeRequestor
	processor envelopeProcessor
	logger    *zap.Logger

	// sem bounds concurrent store requests across all run calls (a single pager
	// instance is shared by every Query), not just within one run.
	sem chan struct{}
}

func newStorePager(requestor storeRequestor, processor envelopeProcessor, logger *zap.Logger) *storePager {
	return &storePager{
		requestor: requestor,
		processor: processor,
		logger:    logger.Named("store-pager"),
		sem:       make(chan struct{}, maxConcurrentStoreRequests),
	}
}

// run fetches [from,to] for pubsubTopic+contentTopics from peerInfo. pageLimit is
// the first page size; shouldProcessNextPage (may be nil) decides per page whether
// to continue and with which size; processEnvelopes controls synchronous handling.
//
// Content-topic chunks are fetched concurrently, EXCEPT when shouldProcessNextPage
// is set: an early-stop callback is stateful (it tracks what it has found and
// mutates the page size), so chunks then run sequentially to keep its semantics
// deterministic. Either way, the pager-wide semaphore bounds total in-flight
// requests.
func (p *storePager) run(
	ctx context.Context,
	peerInfo peer.AddrInfo,
	pubsubTopic string,
	contentTopics []string,
	from, to time.Time,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	chunks := chunkContentTopics(contentTopics, maxContentTopicsPerRequest)

	// Sequential when the caller needs ordered early-stop semantics, or when
	// there is nothing to parallelize.
	if shouldProcessNextPage != nil || len(chunks) <= 1 {
		for _, chunk := range chunks {
			if err := p.fetchChunkBounded(ctx, peerInfo, pubsubTopic, chunk, from, to, pageLimit, shouldProcessNextPage, processEnvelopes); err != nil {
				return err
			}
		}
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for _, chunk := range chunks {
		wg.Add(1)
		go func(contentTopics []string) {
			defer panics.LogOnPanic()
			defer wg.Done()
			if err := p.fetchChunkBounded(ctx, peerInfo, pubsubTopic, contentTopics, from, to, pageLimit, shouldProcessNextPage, processEnvelopes); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel() // stop sibling chunks
				}
				mu.Unlock()
			}
		}(chunk)
	}

	wg.Wait()
	return firstErr
}

// fetchChunkBounded acquires the pager-wide concurrency slot, then paginates the chunk.
func (p *storePager) fetchChunkBounded(
	ctx context.Context,
	peerInfo peer.AddrInfo,
	pubsubTopic string,
	contentTopics []string,
	from, to time.Time,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}
	return p.fetchChunk(ctx, peerInfo, pubsubTopic, contentTopics, from, to, pageLimit, shouldProcessNextPage, processEnvelopes)
}

// fetchChunk paginates one content-topic chunk over [from,to], newest <=24h window
// first, following cursors until each window is exhausted or shouldProcessNextPage
// stops it early.
func (p *storePager) fetchChunk(
	ctx context.Context,
	peerInfo peer.AddrInfo,
	pubsubTopic string,
	contentTopics []string,
	from, to time.Time,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	limit := pageLimit

	for windowEnd := to; windowEnd.After(from); {
		windowStart := from
		if windowEnd.Sub(from) > maxStoreRequestWindow {
			windowStart = windowEnd.Add(-maxStoreRequestWindow)
		}

		var cursor []byte
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			requestID := protocol.GenerateRequestID()
			request := &storepb.StoreQueryRequest{
				RequestId:        hex.EncodeToString(requestID),
				IncludeData:      true,
				PubsubTopic:      proto.String(pubsubTopic),
				ContentTopics:    contentTopics,
				TimeStart:        proto.Int64(windowStart.UnixNano()),
				TimeEnd:          proto.Int64(windowEnd.UnixNano()),
				PaginationCursor: cursor,
				PaginationLimit:  proto.Uint64(limit),
			}

			reqCtx, reqCancel := context.WithTimeout(ctx, storeRequestTimeout)
			messages, nextCursor, err := p.requestor.query(reqCtx, peerInfo, request)
			reqCancel()
			if err != nil {
				return err
			}

			for _, mkv := range messages {
				env := protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic())
				if err := p.processor.OnEnvelope(env, processEnvelopes); err != nil {
					return err
				}
			}

			if shouldProcessNextPage != nil {
				var processNext bool
				processNext, limit = shouldProcessNextPage(len(messages))
				if !processNext {
					return nil
				}
			}

			if nextCursor == nil {
				break // this window is exhausted
			}
			cursor = nextCursor
		}

		windowEnd = windowStart
	}

	return nil
}

func chunkContentTopics(contentTopics []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(contentTopics); i += size {
		end := i + size
		if end > len(contentTopics) {
			end = len(contentTopics)
		}
		chunks = append(chunks, contentTopics[i:end])
	}
	return chunks
}
