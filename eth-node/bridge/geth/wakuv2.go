package gethbridge

import (
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol/pb"
	"github.com/status-im/go-waku/waku/v2/protocol/store"

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

// MinPow returns the PoW value required by this node.
func (w *gethWakuV2Wrapper) MinPow() float64 {
	return 0
}

// MaxMessageSize returns the MaxMessageSize set
func (w *gethWakuV2Wrapper) MaxMessageSize() uint32 {
	return w.waku.MaxMessageSize()
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
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

	if r.Cursor != nil {
		options = append(options, store.WithCursor(&pb.Index{
			Digest:       r.StoreCursor.Digest,
			ReceiverTime: r.StoreCursor.ReceiverTime,
			SenderTime:   r.StoreCursor.SenderTime,
		}))
	}

	var topics []types.TopicType
	for _, topic := range r.Topics {
		topics = append(topics, types.BytesToTopic(topic))
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
		}, nil
	}

	return nil, nil
}

// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
// which is known to implement MailServer interface, and is supposed to process this
// request and respond with a number of peer-to-peer messages (possibly expired),
// which are not supposed to be forwarded any further.
// The whisper protocol is agnostic of the format and contents of envelope.
func (w *gethWakuV2Wrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope types.Envelope, timeout time.Duration) error {
	return errors.New("DEPRECATED")
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
