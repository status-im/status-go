package gethbridge

import (
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/wakuv2"
	wakucommon "github.com/status-im/status-go/wakuv2/common"
)

type gethWakuV2Wrapper struct {
	waku *wakuv2.Waku
}

// NewGethWakuWrapper returns an object that wraps Geth's Waku in a types interface
func NewGethWakuV2Wrapper(w *wakuv2.Waku) types.Waku {
	if w == nil {
		panic("waku cannot be nil")
	}

	return &gethWakuV2Wrapper{
		waku: w,
	}
}

// GetGethWhisperFrom retrieves the underlying whisper Whisper struct from a wrapped Whisper interface
func GetGethWakuV2From(m types.Waku) *wakuv2.Waku {
	return m.(*gethWakuV2Wrapper).waku
}

func (w *gethWakuV2Wrapper) PublicWakuAPI() types.PublicWakuAPI {
	return NewGethPublicWakuV2APIWrapper(wakuv2.NewPublicWakuAPI(w.waku))
}

func (w *gethWakuV2Wrapper) Version() uint {
	return 2
}

func (w *gethWakuV2Wrapper) PeerCount() int {
	return w.waku.PeerCount()
}

// DEPRECATED: Not used in WakuV2
func (w *gethWakuV2Wrapper) MinPow() float64 {
	return 0
}

// MaxMessageSize returns the MaxMessageSize set
func (w *gethWakuV2Wrapper) MaxMessageSize() uint32 {
	return w.waku.MaxMessageSize()
}

// DEPRECATED: not used in WakuV2
func (w *gethWakuV2Wrapper) BloomFilter() []byte {
	return nil
}

// GetCurrentTime returns current time.
func (w *gethWakuV2Wrapper) GetCurrentTime() time.Time {
	return w.waku.CurrentTime()
}

// SetTimeSource assigns a particular source of time to a whisper object.
func (w *gethWakuV2Wrapper) SetTimeSource(timesource func() time.Time) {
	w.waku.SetTimeSource(timesource)
}

func (w *gethWakuV2Wrapper) SubscribeEnvelopeEvents(eventsProxy chan<- types.EnvelopeEvent) types.Subscription {
	events := make(chan wakucommon.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		for e := range events {
			eventsProxy <- *NewWakuV2EnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.waku.SubscribeEnvelopeEvents(events))
}

func (w *gethWakuV2Wrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	return w.waku.GetPrivateKey(id)
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *gethWakuV2Wrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	return w.waku.AddKeyPair(key)
}

// DeleteKeyPair deletes the key with the specified ID if it exists.
func (w *gethWakuV2Wrapper) DeleteKeyPair(keyID string) bool {
	return w.waku.DeleteKeyPair(keyID)
}

func (w *gethWakuV2Wrapper) AddSymKeyDirect(key []byte) (string, error) {
	return w.waku.AddSymKeyDirect(key)
}

func (w *gethWakuV2Wrapper) AddSymKeyFromPassword(password string) (string, error) {
	return w.waku.AddSymKeyFromPassword(password)
}

func (w *gethWakuV2Wrapper) DeleteSymKey(id string) bool {
	return w.waku.DeleteSymKey(id)
}

func (w *gethWakuV2Wrapper) GetSymKey(id string) ([]byte, error) {
	return w.waku.GetSymKey(id)
}

func (w *gethWakuV2Wrapper) Subscribe(opts *types.SubscriptionOptions) (string, error) {
	var (
		err     error
		keyAsym *ecdsa.PrivateKey
		keySym  []byte
	)

	if opts.SymKeyID != "" {
		keySym, err = w.GetSymKey(opts.SymKeyID)
		if err != nil {
			return "", err
		}
	}
	if opts.PrivateKeyID != "" {
		keyAsym, err = w.GetPrivateKey(opts.PrivateKeyID)
		if err != nil {
			return "", err
		}
	}

	f, err := w.createFilterWrapper("", keyAsym, keySym, opts.PoW, opts.Topics)
	if err != nil {
		return "", err
	}

	id, err := w.waku.Subscribe(GetWakuV2FilterFrom(f))
	if err != nil {
		return "", err
	}

	f.(*wakuV2FilterWrapper).id = id
	return id, nil
}

func (w *gethWakuV2Wrapper) GetStats() types.StatsSummary {
	return w.waku.GetStats()
}

func (w *gethWakuV2Wrapper) GetFilter(id string) types.Filter {
	return NewWakuV2FilterWrapper(w.waku.GetFilter(id), id)
}

func (w *gethWakuV2Wrapper) Unsubscribe(id string) error {
	return w.waku.Unsubscribe(id)
}

func (w *gethWakuV2Wrapper) UnsubscribeMany(ids []string) error {
	return w.waku.UnsubscribeMany(ids)
}

func (w *gethWakuV2Wrapper) createFilterWrapper(id string, keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte) (types.Filter, error) {
	return NewWakuV2FilterWrapper(&wakucommon.Filter{
		KeyAsym:  keyAsym,
		KeySym:   keySym,
		Topics:   topics,
		Messages: wakucommon.NewMemoryMessageStore(),
	}, id), nil
}

// DEPRECATED: Not used in waku V2
func (w *gethWakuV2Wrapper) SendMessagesRequest(peerID []byte, r types.MessagesRequest) error {
	return errors.New("DEPRECATED")
}

func (w *gethWakuV2Wrapper) RequestStoreMessages(peerID []byte, r types.MessagesRequest) (*types.StoreRequestCursor, error) {
	var options []store.HistoryRequestOption

	peer, err := peer.Decode(string(peerID))
	if err != nil {
		return nil, err
	}
	options = []store.HistoryRequestOption{
		store.WithPeer(peer),
		store.WithPaging(false, uint64(r.Limit)),
	}

	if r.StoreCursor != nil {
		options = append(options, store.WithCursor(&pb.Index{
			Digest:       r.StoreCursor.Digest,
			ReceiverTime: r.StoreCursor.ReceiverTime,
			SenderTime:   r.StoreCursor.SenderTime,
			PubsubTopic:  r.StoreCursor.PubsubTopic,
		}))
	}

	var topics []wakucommon.TopicType
	for _, topic := range r.Topics {
		topics = append(topics, wakucommon.BytesToTopic(topic))
	}

	pbCursor, err := w.waku.Query(topics, uint64(r.From), uint64(r.To), options)
	if err != nil {
		return nil, err
	}

	if pbCursor != nil {
		return &types.StoreRequestCursor{
			Digest:       pbCursor.Digest,
			ReceiverTime: pbCursor.ReceiverTime,
			SenderTime:   pbCursor.SenderTime,
			PubsubTopic:  pbCursor.PubsubTopic,
		}, nil
	}

	return nil, nil
}

// DEPRECATED: Not used in waku V2
func (w *gethWakuV2Wrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope types.Envelope, timeout time.Duration) error {
	return errors.New("DEPRECATED")
}

func (w *gethWakuV2Wrapper) StartDiscV5() error {
	return w.waku.StartDiscV5()
}

func (w *gethWakuV2Wrapper) StopDiscV5() error {
	return w.waku.StopDiscV5()
}

func (w *gethWakuV2Wrapper) AddStorePeer(address string) (string, error) {
	return w.waku.AddStorePeer(address)
}

func (w *gethWakuV2Wrapper) AddRelayPeer(address string) (string, error) {
	return w.waku.AddRelayPeer(address)
}

func (w *gethWakuV2Wrapper) Peers() map[string][]string {
	return w.waku.Peers()
}

func (w *gethWakuV2Wrapper) DialPeer(address string) error {
	return w.waku.DialPeer(address)
}

func (w *gethWakuV2Wrapper) DialPeerByID(peerID string) error {
	return w.waku.DialPeerByID(peerID)
}

func (w *gethWakuV2Wrapper) DropPeer(peerID string) error {
	return w.waku.DropPeer(peerID)
}

func (w *gethWakuV2Wrapper) ProcessingP2PMessages() bool {
	return w.waku.ProcessingP2PMessages()
}

func (w *gethWakuV2Wrapper) MarkP2PMessageAsProcessed(hash common.Hash) {
	w.waku.MarkP2PMessageAsProcessed(hash)
}

func (w *gethWakuV2Wrapper) SubscribeToConnStatusChanges() (*types.ConnStatusSubscription, error) {
	return w.waku.SubscribeToConnStatusChanges(), nil
}

type wakuV2FilterWrapper struct {
	filter *wakucommon.Filter
	id     string
}

// NewWakuFilterWrapper returns an object that wraps Geth's Filter in a types interface
func NewWakuV2FilterWrapper(f *wakucommon.Filter, id string) types.Filter {
	if f.Messages == nil {
		panic("Messages should not be nil")
	}

	return &wakuV2FilterWrapper{
		filter: f,
		id:     id,
	}
}

// GetWakuFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetWakuV2FilterFrom(f types.Filter) *wakucommon.Filter {
	return f.(*wakuV2FilterWrapper).filter
}

// ID returns the filter ID
func (w *wakuV2FilterWrapper) ID() string {
	return w.id
}
