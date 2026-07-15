package wakuv2

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	commonapi "github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/api/missing"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	common "github.com/status-im/status-go/pkg/messaging/waku/common"
	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// Hash-first spectate backfill (issue #21470-hf).
//
// A keyless spectator's store-node backfill of a community's single universal content
// topic re-downloads the same 1.4MB community description dozens of times: each
// heartbeat republish is a fresh ciphertext with a unique envelope hash, so post-download
// dedup cannot collapse them. Device measurement (Samsung S21FE) put the useful content
// at ~1/1000th of the bytes ingested.
//
// go-waku's own missing-message verifier already solves the shape of this: it queries the
// store for hashes+metadata ONLY (no bodies), checks each hash against what it already
// holds, and fetches full bodies for the unknown hashes alone
// (go-waku waku/v2/api/missing/missing_messages.go). We apply the same hash-first shape
// to the spectate backfill so bodies are fetched only for envelopes we have not already
// seen. HistoryRetriever.Query cannot be reused because it hardcodes IncludeData:true
// (go-waku waku/v2/api/history/history.go), so this path drives the store client directly
// via the same requestor/message-tracker the verifier uses.
const (
	// hashFirstMetadataPageSize bounds the metadata-only (hash) page walk. Mirrors the
	// verifier's messageFetchPageSize.
	hashFirstMetadataPageSize = 100

	// hashFirstMaxHashesPerRequest bounds how many hashes a single body-fetch store query
	// may carry. Mirrors the verifier's maxMsgHashesPerRequest — the store node caps the
	// number of message_hashes keys per request.
	hashFirstMaxHashesPerRequest = 50

	// hashFirstMaxContentTopicsPerRequest bounds how many content topics a single metadata
	// store query may carry. Mirrors the classic HistoryRetriever's maxTopicsPerRequest and
	// the verifier's maxContentTopicsPerRequest — the store node caps the number of content
	// topics per request. The general (all-filters) history sync batches many more topics
	// than the spectate path, so the metadata walk must chunk them (issue #21470-hf).
	hashFirstMaxContentTopicsPerRequest = 10

	// hashFirstIngestHighWater bounds how many fetched-but-not-yet-consumed envelopes
	// (decrypted messages sitting in the per-filter stores awaiting the retrieve loop) the
	// body fetch lets accumulate before pausing. Without this the fetch outruns the slow
	// per-description consumer and the backlog balloons memory: at ~1.4MB per
	// community-description envelope, letting the msgQueue fill to its 1024 cap is ~1.4GB.
	// Capping the in-flight backlog here keeps the transient peak an order of magnitude
	// lower while still pipelining fetch and ingest (issue #21470-hf).
	hashFirstIngestHighWater = 128
)

// hashFirstBackpressurePoll is how often a paused body fetch re-checks the ingest backlog.
// This is flow control, not a hot path, so a coarse interval keeps the poll cost negligible
// while still resuming promptly once the consumer drains (issue #21470-hf).
const hashFirstBackpressurePoll = 100 * time.Millisecond

// hashFirstMaxWindow caps the duration of a single metadata store query. The classic
// HistoryRetriever caps each query to 24h and walks a longer range backward (see
// history.go's exceeds24h handling); the general history sync can span the ~9-day default
// sync period, so the metadata walk is split into <=24h sub-windows to match (issue
// #21470-hf).
const hashFirstMaxWindow = 24 * time.Hour

// hashFirstTimeWindow is one <=24h sub-window of the metadata walk, in unix nanoseconds.
type hashFirstTimeWindow struct {
	fromNano int64
	toNano   int64
}

// partitionContentTopics splits content topics into chunks of at most maxPerRequest, so
// each metadata store query stays within the store node's per-request content-topic limit.
// A non-positive maxPerRequest yields a single chunk (defensive: never emit an unbounded
// or infinite split). Order is preserved.
func partitionContentTopics(topics []string, maxPerRequest int) [][]string {
	if len(topics) == 0 {
		return nil
	}
	if maxPerRequest <= 0 {
		return [][]string{topics}
	}
	var chunks [][]string
	for i := 0; i < len(topics); i += maxPerRequest {
		j := i + maxPerRequest
		if j > len(topics) {
			j = len(topics)
		}
		chunks = append(chunks, topics[i:j])
	}
	return chunks
}

// splitWindowNewestFirst splits [fromNano, toNano) into consecutive sub-windows of at most
// maxWindowNano, ordered NEWEST-FIRST (the first window ends at toNano). This mirrors the
// classic HistoryRetriever, which caps each store query to 24h and walks a longer range
// backward: a single metadata query over a multi-day range would silently cover only the
// newest window while the watermark still advances to `to`, losing the older days.
// Newest-first ordering keeps bodies fetched most-recent-first (matching aa9fa29b5). An
// empty or inverted range yields no windows; a non-positive maxWindowNano yields a single
// window spanning the range (defensive: never loop forever).
func splitWindowNewestFirst(fromNano, toNano, maxWindowNano int64) []hashFirstTimeWindow {
	if toNano <= fromNano {
		return nil
	}
	if maxWindowNano <= 0 {
		return []hashFirstTimeWindow{{fromNano: fromNano, toNano: toNano}}
	}
	var windows []hashFirstTimeWindow
	end := toNano
	for end > fromNano {
		start := end - maxWindowNano
		if start < fromNano {
			start = fromNano
		}
		windows = append(windows, hashFirstTimeWindow{fromNano: start, toNano: end})
		end = start
	}
	return windows
}

// partitionMessageHashes splits hashes into batches of at most maxPerRequest, so each
// body-fetch store query stays within the store node's per-request hash-key limit. A
// non-positive maxPerRequest yields a single batch (defensive: never emit an unbounded
// or infinite split).
func partitionMessageHashes(hashes []wpb.MessageHash, maxPerRequest int) [][]wpb.MessageHash {
	if len(hashes) == 0 {
		return nil
	}
	if maxPerRequest <= 0 {
		return [][]wpb.MessageHash{hashes}
	}
	var batches [][]wpb.MessageHash
	for i := 0; i < len(hashes); i += maxPerRequest {
		j := i + maxPerRequest
		if j > len(hashes) {
			j = len(hashes)
		}
		batches = append(batches, hashes[i:j])
	}
	return batches
}

// filterUnknownHashes returns the hashes not already held locally, in first-seen order
// and de-duplicated, together with how many inputs were already known. exists is the
// local "already processed this envelope" authority (the verifier's message tracker).
// A duplicate hash within the input is counted once and, if unknown, fetched once.
func filterUnknownHashes(hashes []wpb.MessageHash, exists func(wpb.MessageHash) (bool, error)) (unknown []wpb.MessageHash, known int, err error) {
	seen := make(map[wpb.MessageHash]struct{}, len(hashes))
	for _, h := range hashes {
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}

		has, err := exists(h)
		if err != nil {
			return nil, 0, err
		}
		if has {
			known++
			continue
		}
		unknown = append(unknown, h)
	}
	return unknown, known, nil
}

// buildMetadataQuery builds the metadata-only (hash) store query that walks the window.
// It requests NO bodies (IncludeData:false) and NEWEST-FIRST pagination
// (PaginationForward:false) so bodies are later fetched most-recent-first — for a keyed
// fetch that means a cancellation (leave/background) keeps the most useful (newest)
// history rather than the oldest (issue #21470-hf enhancement).
func buildMetadataQuery(requestID string, pubsubTopic string, contentTopics []string, fromNano, toNano int64) *storepb.StoreQueryRequest {
	return &storepb.StoreQueryRequest{
		RequestId:         requestID,
		IncludeData:       false,
		PubsubTopic:       &pubsubTopic,
		ContentTopics:     contentTopics,
		TimeStart:         proto.Int64(fromNano),
		TimeEnd:           proto.Int64(toNano),
		PaginationForward: false,
		PaginationLimit:   proto.Uint64(hashFirstMetadataPageSize),
	}
}

// planBodyFetch decides which unknown-hash bodies to actually fetch. When skipBodies is
// set (keyless spectator with a resolved description — every body here is undecryptable),
// nothing is fetched and all unknowns are counted as skipped; the hashes were still
// walked, so the watermark advances and the skip is measurable. Otherwise every unknown
// hash is fetched (issue #21470-hf enhancement).
func planBodyFetch(unknown []wpb.MessageHash, skipBodies bool) (toFetch []wpb.MessageHash, skippedKeyless int) {
	if skipBodies {
		return nil, len(unknown)
	}
	return unknown, 0
}

// waitForIngestCapacity blocks the body fetch until the ingest backlog reported by depth
// drops below highWater, polling every poll. It returns immediately (paused=false) when
// highWater <= 0 (disabled) or the backlog is already below it. While paused it honours
// ctx cancellation, returning the ctx error so leave/background aborts promptly. This is
// pure flow control: it holds no locks while sleeping, so the (separate) retrieve-loop and
// decrypt goroutines keep draining the backlog and it always makes progress unless the
// caller is cancelled (issue #21470-hf).
func waitForIngestCapacity(ctx context.Context, depth func() int, highWater int, poll time.Duration) (paused bool, err error) {
	if highWater <= 0 || depth() < highWater {
		return false, nil
	}
	paused = true
	for {
		if err := ctx.Err(); err != nil {
			return paused, err
		}
		select {
		case <-ctx.Done():
			return paused, ctx.Err()
		case <-time.After(poll):
		}
		if depth() < highWater {
			return paused, nil
		}
	}
}

// pendingIngestCount is the body-fetch backpressure signal: the number of decrypted
// envelopes waiting in the filter stores for the retrieve loop to consume (issue #21470-hf).
func (w *Waku) pendingIngestCount() int {
	return w.filters.PendingMessageCount()
}

// ProcessMailserverBatchHashFirst backfills a store-node batch hash-first: it walks the
// window pulling only message hashes+metadata, filters out envelopes already held
// locally, then fetches full bodies only for the unknown hashes and feeds them into the
// normal ingest path (OnNewEnvelopes), exactly as the classic full-body path does. It
// returns per-batch stats for the once-per-backfill INFO log.
//
// The metadata walk generalizes to the general (all-filters) history sync's batch shapes
// (issue #21470-hf): content topics are chunked to at most hashFirstMaxContentTopicsPerRequest
// per store query, and a multi-day window is split into <=24h sub-windows walked
// newest-first, both mirroring the classic HistoryRetriever so coverage is identical and no
// day is silently dropped from a range wider than 24h.
//
// Cancellation: the caller's ctx is threaded into every store query and checked between
// chunks, windows, pages and body-fetch partitions, so leave/background aborts promptly. As
// with the classic path, watermark bookkeeping lives in the caller and advances only after
// this returns without error, so a cancelled batch marks nothing fetched.
func (w *Waku) ProcessMailserverBatchHashFirst(
	ctx context.Context,
	batch types.MailserverBatch,
	storenode peer.AddrInfo,
	processEnvelopes bool,
	skipBodies bool,
) (types.HashFirstStats, error) {
	var stats types.HashFirstStats

	pubsubTopic := w.GetPubsubTopic(batch.PubsubTopic)
	contentTopics := make([]string, 0, len(batch.Topics))
	for _, topic := range batch.Topics {
		contentTopics = append(contentTopics, common.BytesToTopic(topic.Bytes()).ContentTopic())
	}

	requestor := missing.NewDefaultStorenodeRequestor(w.node.Store())

	// Phase 1 — metadata-only page walk (newest-first): collect every hash in the window,
	// skipping those already held locally. The general sync batches many topics over a
	// multi-day window, so the walk is chunked by content topic (store per-request cap) and
	// by <=24h sub-window (query duration cap), exactly as the classic HistoryRetriever does.
	windows := splitWindowNewestFirst(batch.From.UnixNano(), batch.To.UnixNano(), hashFirstMaxWindow.Nanoseconds())
	var unknownHashes []wpb.MessageHash
	for _, topicChunk := range partitionContentTopics(contentTopics, hashFirstMaxContentTopicsPerRequest) {
		for _, window := range windows {
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			chunkUnknown, err := w.walkMetadataWindow(ctx, requestor, storenode, pubsubTopic, topicChunk, window, &stats)
			if err != nil {
				return stats, err
			}
			unknownHashes = append(unknownHashes, chunkUnknown...)
		}
	}

	// De-duplicate across chunks, windows and pages: a hash unknown in one place must not be
	// fetched twice if the store repeats it.
	unknownHashes, _, err := filterUnknownHashes(unknownHashes, func(wpb.MessageHash) (bool, error) { return false, nil })
	if err != nil {
		return stats, err
	}

	// Keyless spectator with a resolved description: every remaining body on the shared
	// community content topic is undecryptable, so skip the body fetch entirely (the
	// hashes were still walked, so the watermark advances and the skip is measured).
	toFetch, skippedKeyless := planBodyFetch(unknownHashes, skipBodies)
	stats.BodiesSkippedKeyless += skippedKeyless

	// Phase 2 — body fetch: pull full bodies ONLY for the unknown hashes, in store-capped
	// batches, and feed them into the normal ingest path.
	for _, part := range partitionMessageHashes(toFetch, hashFirstMaxHashesPerRequest) {
		if err := ctx.Err(); err != nil {
			return stats, err
		}

		hashBytes := make([][]byte, 0, len(part))
		for _, h := range part {
			hashBytes = append(hashBytes, h.Bytes())
		}
		bodyQuery := &storepb.StoreQueryRequest{
			RequestId:       hex.EncodeToString(protocol.GenerateRequestID()),
			IncludeData:     true,
			MessageHashes:   hashBytes,
			PaginationLimit: proto.Uint64(hashFirstMaxHashesPerRequest),
		}

		bodyResult, err := requestor.Query(ctx, storenode, bodyQuery)
		if err != nil {
			return stats, err
		}
		for {
			// Backpressure: before enqueuing another page of (large) decrypted bodies,
			// wait for the ingest backlog to fall below the high-water mark so fetched
			// envelopes cannot pile up faster than the slow per-description consumer
			// drains them (issue #21470-hf).
			paused, err := waitForIngestCapacity(ctx, w.pendingIngestCount, hashFirstIngestHighWater, hashFirstBackpressurePoll)
			if err != nil {
				return stats, err
			}
			if paused {
				stats.BodyFetchThrottled++
				w.logger.Debug("hash-first body fetch throttled: ingest backlog above high-water",
					zap.Int("highWater", hashFirstIngestHighWater),
					zap.Int("pending", w.pendingIngestCount()))
			}
			if err := w.ingestBodyPage(bodyResult, processEnvelopes, &stats); err != nil {
				return stats, err
			}
			if bodyResult.IsComplete() {
				break
			}
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			if err := bodyResult.Next(ctx); err != nil {
				return stats, err
			}
		}
	}

	return stats, nil
}

// walkMetadataWindow runs the metadata-only page walk for one content-topic chunk over one
// <=24h sub-window: it pages newest-first pulling only hashes, accumulates HashesSeen /
// HashesKnown into stats, and returns the hashes not already held locally (per-page
// de-duplicated; the caller de-duplicates across chunks/windows). The ctx is checked
// between pages so a cancel aborts promptly.
func (w *Waku) walkMetadataWindow(
	ctx context.Context,
	requestor commonapi.StorenodeRequestor,
	storenode peer.AddrInfo,
	pubsubTopic string,
	contentTopics []string,
	window hashFirstTimeWindow,
	stats *types.HashFirstStats,
) ([]wpb.MessageHash, error) {
	metadataQuery := buildMetadataQuery(
		hex.EncodeToString(protocol.GenerateRequestID()),
		pubsubTopic, contentTopics, window.fromNano, window.toNano)

	result, err := requestor.Query(ctx, storenode, metadataQuery)
	if err != nil {
		return nil, err
	}

	var unknownHashes []wpb.MessageHash
	for {
		pageHashes := make([]wpb.MessageHash, 0, len(result.Messages()))
		for _, mkv := range result.Messages() {
			stats.HashesSeen++
			pageHashes = append(pageHashes, wpb.ToMessageHash(mkv.MessageHash))
		}
		pageUnknown, known, err := filterUnknownHashes(pageHashes, w.MessageExists)
		if err != nil {
			return nil, err
		}
		stats.HashesKnown += known
		unknownHashes = append(unknownHashes, pageUnknown...)

		if result.IsComplete() {
			break
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := result.Next(ctx); err != nil {
			return nil, err
		}
	}
	return unknownHashes, nil
}

// ingestBodyPage feeds one page of fetched full-body envelopes into the normal ingest
// path (OnNewEnvelopes, StoreMessageType) and accumulates fetched-body stats.
func (w *Waku) ingestBodyPage(result commonapi.StoreRequestResult, processEnvelopes bool, stats *types.HashFirstStats) error {
	for _, mkv := range result.Messages() {
		if mkv.Message == nil {
			continue
		}
		envelope := protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic())
		if err := w.OnNewEnvelopes(envelope, common.StoreMessageType, processEnvelopes); err != nil {
			return err
		}
		stats.BodiesFetched++
		stats.BytesEstimate += int64(proto.Size(mkv.Message))
	}
	return nil
}
