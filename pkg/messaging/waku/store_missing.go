package wakuv2

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

const (
	// maxMessageHashesPerRequest is the store protocol's per-request hash limit for
	// by-hash body fetches.
	maxMessageHashesPerRequest = 50
	// missingMessagePageSize is the pagination limit for missing-message queries.
	missingMessagePageSize = 100
)

// FetchMissingMessages reconciles a window against the store: it lists the message
// hashes the store holds for [batch.From, batch.To] on the batch's topics, fetches
// the bodies of the ones not already known locally (messageExists), and pushes
// them through process. It pins ONE storenode for the whole operation — hash
// listing, body fetch, and all of their pagination — so every page comes from the
// same node and cursors stay valid; it fails over to another node only if that one
// fails the whole operation. messageExists and process are supplied by the waku
// adapter (local dedup and the received-envelope sink, respectively).
func (sc *StoreClient) FetchMissingMessages(
	ctx context.Context,
	batch types.MailserverBatch,
	messageExists func(pb.MessageHash) (bool, error),
	process func(*protocol.Envelope) error,
) error {
	pubsubTopic := sc.resolvePubsubTopic(batch.PubsubTopic)
	contentTopics := sc.contentTopics(batch)
	if len(contentTopics) == 0 {
		return nil
	}

	return sc.withStorenode(ctx, func(ctx context.Context, node peer.AddrInfo) error {
		missing, err := sc.listMissingHashes(ctx, node, pubsubTopic, contentTopics, batch.From, batch.To, messageExists)
		if err != nil {
			return err
		}
		if len(missing) == 0 {
			return nil
		}
		return sc.fetchHashes(ctx, node, missing, process)
	})
}

// listMissingHashes queries the store for the message hashes in [from,to] on the
// given content topics (chunked) and returns the ones not already known locally.
// It requests hashes only (IncludeData=false) to keep the diff cheap.
func (sc *StoreClient) listMissingHashes(
	ctx context.Context,
	node peer.AddrInfo,
	pubsubTopic string,
	contentTopics []string,
	from, to time.Time,
	messageExists func(pb.MessageHash) (bool, error),
) ([][]byte, error) {
	var missing [][]byte
	for _, chunk := range chunkContentTopics(contentTopics, maxContentTopicsPerRequest) {
		var cursor []byte
		for {
			request := &storepb.StoreQueryRequest{
				RequestId:        hex.EncodeToString(protocol.GenerateRequestID()),
				IncludeData:      false,
				PubsubTopic:      proto.String(pubsubTopic),
				ContentTopics:    chunk,
				TimeStart:        proto.Int64(from.UnixNano()),
				TimeEnd:          proto.Int64(to.UnixNano()),
				PaginationCursor: cursor,
				PaginationLimit:  proto.Uint64(missingMessagePageSize),
			}

			reqCtx, cancel := context.WithTimeout(ctx, storeRequestTimeout)
			messages, next, err := sc.requestor.query(reqCtx, node, request)
			cancel()
			if err != nil {
				return nil, err
			}

			for _, mkv := range messages {
				exists, err := messageExists(pb.ToMessageHash(mkv.MessageHash))
				if err != nil {
					return nil, err
				}
				if !exists {
					missing = append(missing, mkv.MessageHash)
				}
			}

			if next == nil {
				break
			}
			cursor = next
		}
	}
	return missing, nil
}

// fetchHashes fetches the bodies of the given message hashes (IncludeData=true)
// and pushes each through process.
func (sc *StoreClient) fetchHashes(
	ctx context.Context,
	node peer.AddrInfo,
	hashes [][]byte,
	process func(*protocol.Envelope) error,
) error {
	for _, chunk := range chunkHashes(hashes, maxMessageHashesPerRequest) {
		var cursor []byte
		for {
			request := &storepb.StoreQueryRequest{
				RequestId:        hex.EncodeToString(protocol.GenerateRequestID()),
				IncludeData:      true,
				MessageHashes:    chunk,
				PaginationCursor: cursor,
				PaginationLimit:  proto.Uint64(maxMessageHashesPerRequest),
			}

			reqCtx, cancel := context.WithTimeout(ctx, storeRequestTimeout)
			messages, next, err := sc.requestor.query(reqCtx, node, request)
			cancel()
			if err != nil {
				return err
			}

			for _, mkv := range messages {
				env := protocol.NewEnvelope(mkv.Message, mkv.Message.GetTimestamp(), mkv.GetPubsubTopic())
				if err := process(env); err != nil {
					return err
				}
			}

			if next == nil {
				break
			}
			cursor = next
		}
	}
	return nil
}

func chunkHashes(hashes [][]byte, size int) [][][]byte {
	var chunks [][][]byte
	for i := 0; i < len(hashes); i += size {
		end := i + size
		if end > len(hashes) {
			end = len(hashes)
		}
		chunks = append(chunks, hashes[i:end])
	}
	return chunks
}
