package wakuv2

import (
	"context"
	"encoding/hex"

	"github.com/libp2p/go-libp2p/core/peer"
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
)

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

// ProcessMailserverBatchHashFirst backfills a store-node batch hash-first: it walks the
// window pulling only message hashes+metadata, filters out envelopes already held
// locally, then fetches full bodies only for the unknown hashes and feeds them into the
// normal ingest path (OnNewEnvelopes), exactly as the classic full-body path does. It
// returns per-batch stats for the once-per-backfill INFO log.
//
// Cancellation: the caller's ctx is threaded into every store query and checked between
// pages and between body-fetch partitions, so leave/background aborts promptly. As with
// the classic path, watermark bookkeeping lives in the caller and advances only after
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

	// Phase 1 — metadata-only page walk: collect every hash in the window, skipping those
	// already held locally.
	var unknownHashes []wpb.MessageHash
	metadataQuery := &storepb.StoreQueryRequest{
		RequestId:       hex.EncodeToString(protocol.GenerateRequestID()),
		IncludeData:     false,
		PubsubTopic:     &pubsubTopic,
		ContentTopics:   contentTopics,
		TimeStart:       proto.Int64(batch.From.UnixNano()),
		TimeEnd:         proto.Int64(batch.To.UnixNano()),
		PaginationLimit: proto.Uint64(hashFirstMetadataPageSize),
	}

	result, err := requestor.Query(ctx, storenode, metadataQuery)
	if err != nil {
		return stats, err
	}
	for {
		pageHashes := make([]wpb.MessageHash, 0, len(result.Messages()))
		for _, mkv := range result.Messages() {
			stats.HashesSeen++
			pageHashes = append(pageHashes, wpb.ToMessageHash(mkv.MessageHash))
		}
		pageUnknown, known, err := filterUnknownHashes(pageHashes, w.MessageExists)
		if err != nil {
			return stats, err
		}
		stats.HashesKnown += known
		unknownHashes = append(unknownHashes, pageUnknown...)

		if result.IsComplete() {
			break
		}
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if err := result.Next(ctx); err != nil {
			return stats, err
		}
	}

	// De-duplicate across pages as well: a hash unknown on one page must not be fetched
	// twice if the store repeats it.
	unknownHashes, _, err = filterUnknownHashes(unknownHashes, func(wpb.MessageHash) (bool, error) { return false, nil })
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
