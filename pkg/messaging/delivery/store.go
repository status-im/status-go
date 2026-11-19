package delivery

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	kcommon "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wakupb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// storeStatusOK is the store protocol's success status code.
const storeStatusOK = 200

// StoreQuery retrieves historic messages for a single batch from the store
// nodes the node reports for its preset.
func (a *Adapter) StoreQuery(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return a.storeClient.Query(ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
}

// FetchMessagesByHashes retrieves the messages with the given hex hashes from
// the store nodes, with the same failover as StoreQuery, and reports each as
// received.
func (a *Adapter) FetchMessagesByHashes(ctx context.Context, messageHashes []string) error {
	hashes := make([][]byte, 0, len(messageHashes))
	for _, h := range messageHashes {
		decoded, err := hexutil.Decode(h)
		if err != nil {
			a.logger.Debug("invalid message hash for storenode fetch", zap.String("messageHash", h), zap.Error(err))
			continue
		}
		hashes = append(hashes, decoded)
	}
	if len(hashes) == 0 {
		return nil
	}

	return a.storeClient.FetchByHashes(ctx, func(nodeCtx context.Context, storenode peer.AddrInfo) error {
		request := &storepb.StoreQueryRequest{
			RequestId:         uuid.NewString(),
			IncludeData:       true,
			MessageHashes:     hashes,
			PaginationForward: true,
			PaginationLimit:   proto.Uint64(uint64(len(hashes))),
		}
		for {
			messages, cursor, err := a.storeRequestor.Query(nodeCtx, storenode, request)
			if err != nil {
				return err
			}
			for _, kv := range messages {
				a.reportStored(kv)
			}
			if len(cursor) == 0 {
				return nil
			}
			request.PaginationCursor = cursor
		}
	})
}

func (a *Adapter) reportStored(kv *storepb.WakuMessageKeyValue) {
	msg := kv.GetMessage()
	if msg == nil {
		return
	}
	a.envelopeFeed.Send(types.EnvelopeEvent{
		Event: types.EventEnvelopeAvailable,
		Hash:  cryptotypes.BytesToHash(kv.GetMessageHash()),
		Data: &types.ReceivedMessage{
			Hash:         kv.GetMessageHash(),
			ContentTopic: msg.ContentTopic,
			Payload:      msg.Payload,
			Ephemeral:    msg.GetEphemeral(),
			Meta:         msg.Meta,
			PubsubTopic:  kv.GetPubsubTopic(),
			Version:      msg.GetVersion(),
			Timestamp:    msg.GetTimestamp(),
		},
	})
}

// nodeStoreRequestor runs the store client's requests through the node's
// store query.
type nodeStoreRequestor struct {
	mc *messaging.MessagingClient
}

func (r nodeStoreRequestor) Query(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	response, err := r.mc.Node().Store().Query(ctx, toKernelStoreRequest(request), peerInfo)
	if err != nil {
		return nil, nil, err
	}
	if response.StatusCode != nil && *response.StatusCode != storeStatusOK {
		return nil, nil, fmt.Errorf("delivery: store query: status %d: %s", *response.StatusCode, response.StatusDesc)
	}
	return fromKernelStoreResponse(response)
}

// storeProcessor reports the messages the store client fetched as received.
type storeProcessor struct {
	a *Adapter
}

func (p storeProcessor) OnEnvelope(env *protocol.Envelope, _ bool) error {
	p.a.reportStored(&storepb.WakuMessageKeyValue{
		MessageHash: env.Hash().Bytes(),
		Message:     env.Message(),
		PubsubTopic: proto.String(env.PubsubTopic()),
	})
	return nil
}

func toKernelStoreRequest(r *storepb.StoreQueryRequest) *kcommon.StoreQueryRequest {
	req := &kcommon.StoreQueryRequest{
		RequestId:         r.RequestId,
		IncludeData:       r.IncludeData,
		TimeStart:         r.TimeStart,
		TimeEnd:           r.TimeEnd,
		PaginationForward: r.PaginationForward,
		PaginationLimit:   r.PaginationLimit,
	}
	if r.PubsubTopic != nil {
		req.PubsubTopic = *r.PubsubTopic
	}
	if len(r.ContentTopics) > 0 {
		topics := append([]string(nil), r.ContentTopics...)
		req.ContentTopics = &topics
	}
	if len(r.MessageHashes) > 0 {
		hashes := make([]kcommon.MessageHash, 0, len(r.MessageHashes))
		for _, h := range r.MessageHashes {
			hashes = append(hashes, kcommon.MessageHash(hexutil.Encode(h)))
		}
		req.MessageHashes = &hashes
	}
	if len(r.PaginationCursor) > 0 {
		cursor := kcommon.MessageHash(hexutil.Encode(r.PaginationCursor))
		req.PaginationCursor = &cursor
	}
	return req
}

func fromKernelStoreResponse(r *kcommon.StoreQueryResponse) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	var messages []*storepb.WakuMessageKeyValue
	if r.Messages != nil {
		messages = make([]*storepb.WakuMessageKeyValue, 0, len(*r.Messages))
		for _, m := range *r.Messages {
			if m.WakuMessage == nil {
				continue
			}
			hash, err := hexutil.Decode(string(m.MessageHash))
			if err != nil {
				return nil, nil, fmt.Errorf("delivery: store query: message hash: %w", err)
			}
			messages = append(messages, &storepb.WakuMessageKeyValue{
				MessageHash: hash,
				PubsubTopic: proto.String(m.PubsubTopic),
				Message: &wakupb.WakuMessage{
					Payload:        m.WakuMessage.Payload,
					ContentTopic:   m.WakuMessage.ContentTopic,
					Version:        m.WakuMessage.Version,
					Timestamp:      m.WakuMessage.Timestamp,
					Meta:           m.WakuMessage.Meta,
					Ephemeral:      m.WakuMessage.Ephemeral,
					RateLimitProof: m.WakuMessage.RateLimitProof,
				},
			})
		}
	}
	var cursor []byte
	if r.PaginationCursor != "" {
		c, err := hexutil.Decode(string(r.PaginationCursor))
		if err != nil {
			return nil, nil, fmt.Errorf("delivery: store query: cursor: %w", err)
		}
		cursor = c
	}
	return messages, cursor, nil
}
