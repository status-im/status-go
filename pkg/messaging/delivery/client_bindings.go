package delivery

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/protobuf/proto"

	kcommon "github.com/logos-messaging/logos-delivery-go-bindings/pkg/kernel/common"
	"github.com/logos-messaging/logos-delivery-go-bindings/pkg/messaging"
	wakupb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"

	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// bindingsClient adapts the bindings' MessagingClient to Client, translating
// its events into this package's.
type bindingsClient struct {
	mc     *messaging.MessagingClient
	events chan Event
}

// NewClient creates a node from cfg. It is not started.
func NewClient(cfg Config) (Client, error) {
	mc, err := messaging.New(bindingsConfig(cfg))
	if err != nil {
		return nil, err
	}
	c := &bindingsClient{mc: mc, events: make(chan Event, cap(mc.Events()))}
	go c.forward()
	return c, nil
}

func bindingsConfig(cfg Config) messaging.Config {
	overrides := messaging.Overrides{}
	if cfg.NodeKey != nil {
		overrides["nodekey"] = hex.EncodeToString(crypto.FromECDSA(cfg.NodeKey))
	}
	if cfg.TCPPort != 0 {
		overrides["tcp-port"] = cfg.TCPPort
	}
	if cfg.UDPPort != 0 {
		overrides["discv5-udp-port"] = cfg.UDPPort
	}
	if cfg.DataDir != "" {
		overrides["local-storage-path"] = cfg.DataDir
	}
	for k, v := range cfg.Overrides {
		overrides[k] = v
	}
	if len(overrides) == 0 {
		overrides = nil
	}
	return messaging.Config{
		Mode:               messaging.Mode(cfg.Mode),
		Preset:             cfg.Preset,
		MessagingOverrides: overrides,
	}
}

func (c *bindingsClient) forward() {
	defer panics.LogOnPanic()
	defer close(c.events)
	for ev := range c.mc.Events() {
		if e := translate(ev); e != nil {
			c.events <- e
		}
	}
}

func translate(ev messaging.Event) Event {
	switch e := ev.(type) {
	case messaging.MessageReceivedEvent:
		return ReceivedEvent{
			MessageHash:  e.MessageHash,
			ContentTopic: e.Message.ContentTopic,
			Payload:      e.Message.Payload,
			Meta:         e.Message.Meta,
			Version:      e.Message.Version,
			Timestamp:    e.Message.Timestamp,
			Ephemeral:    e.Message.Ephemeral,
		}
	case messaging.MessageSentEvent:
		return SentEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash}
	case messaging.MessagePropagatedEvent:
		return PropagatedEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash}
	case messaging.MessageErrorEvent:
		return ErrorEvent{RequestID: string(e.RequestID), MessageHash: e.MessageHash, Err: e.Err}
	case messaging.ConnectionStatusEvent:
		return ConnectionEvent{Status: ConnectionStatus(e.Status)}
	}
	return nil
}

func (c *bindingsClient) Start() error         { return c.mc.Start() }
func (c *bindingsClient) Stop() error          { return c.mc.Stop() }
func (c *bindingsClient) Close() error         { return c.mc.Close() }
func (c *bindingsClient) Events() <-chan Event { return c.events }

func (c *bindingsClient) Subscribe(contentTopic string) error {
	return c.mc.Subscribe(contentTopic)
}

func (c *bindingsClient) Unsubscribe(contentTopic string) error {
	return c.mc.Unsubscribe(contentTopic)
}

func (c *bindingsClient) Send(ctx context.Context, contentTopic string, payload []byte, ephemeral bool) (string, error) {
	id, err := c.mc.Send(ctx, contentTopic, payload, ephemeral)
	return string(id), err
}

func (c *bindingsClient) StoreQuery(ctx context.Context, peerInfo peer.AddrInfo, request *storepb.StoreQueryRequest) ([]*storepb.WakuMessageKeyValue, []byte, error) {
	response, err := c.mc.Node().Store().Query(ctx, toKernelStoreRequest(request), peerInfo)
	if err != nil {
		return nil, nil, err
	}
	if response.StatusCode != nil && *response.StatusCode != storeStatusOK {
		return nil, nil, fmt.Errorf("delivery: store query: status %d: %s", *response.StatusCode, response.StatusDesc)
	}
	return fromKernelStoreResponse(response)
}

func (c *bindingsClient) ConnectedPeers() (types.PeerStats, error) {
	peers, err := c.mc.Node().Peers().ConnectedInfo()
	if err != nil {
		return nil, err
	}
	stats := make(types.PeerStats, len(peers))
	for id, info := range peers {
		stats[id] = types.WakuV2Peer{Protocols: info.Protocols, Addresses: info.Addresses}
	}
	return stats, nil
}

func (c *bindingsClient) PeerID() (peer.ID, error) { return c.mc.Node().Debug().PeerID() }
func (c *bindingsClient) Metrics() (string, error) { return c.mc.Node().Debug().Metrics() }

// storeStatusOK is the store protocol's success status code.
const storeStatusOK = 200

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
			hash, err := hexutil.Decode(string(m.MessageHash))
			if err != nil {
				return nil, nil, fmt.Errorf("delivery: store query: message hash: %w", err)
			}
			kv := &storepb.WakuMessageKeyValue{MessageHash: hash, PubsubTopic: proto.String(m.PubsubTopic)}
			if m.WakuMessage != nil {
				kv.Message = &wakupb.WakuMessage{
					Payload:        m.WakuMessage.Payload,
					ContentTopic:   m.WakuMessage.ContentTopic,
					Version:        m.WakuMessage.Version,
					Timestamp:      m.WakuMessage.Timestamp,
					Meta:           m.WakuMessage.Meta,
					Ephemeral:      m.WakuMessage.Ephemeral,
					RateLimitProof: m.WakuMessage.RateLimitProof,
				}
			}
			messages = append(messages, kv)
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
