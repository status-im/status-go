package gethbridge

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
)

type gethWakuWrapper struct {
	waku *waku.Waku
}

// NewGethWakuWrapper returns an object that wraps Geth's Waku in a types interface
func NewGethWakuWrapper(w *waku.Waku) types.Waku {
	if w == nil {
		panic("waku cannot be nil")
	}

	return &gethWakuWrapper{
		waku: w,
	}
}

// GetGethWhisperFrom retrieves the underlying whisper Whisper struct from a wrapped Whisper interface
func GetGethWakuFrom(m types.Waku) *waku.Waku {
	return m.(*gethWakuWrapper).waku
}

func (w *gethWakuWrapper) PublicWakuAPI() types.PublicWakuAPI {
	return NewGethPublicWakuAPIWrapper(waku.NewPublicWakuAPI(w.waku))
}

// MinPow returns the PoW value required by this node.
func (w *gethWakuWrapper) MinPow() float64 {
	return w.waku.MinPow()
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (w *gethWakuWrapper) BloomFilter() []byte {
	return w.waku.BloomFilter()
}

// GetCurrentTime returns current time.
func (w *gethWakuWrapper) GetCurrentTime() time.Time {
	return w.waku.CurrentTime()
}

// SetTimeSource assigns a particular source of time to a whisper object.
func (w *gethWakuWrapper) SetTimeSource(timesource func() time.Time) {
	w.waku.SetTimeSource(timesource)
}

func (w *gethWakuWrapper) SubscribeEnvelopeEvents(eventsProxy chan<- types.EnvelopeEvent) types.Subscription {
	events := make(chan wakucommon.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		for e := range events {
			eventsProxy <- *NewWakuEnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.waku.SubscribeEnvelopeEvents(events))
}

func (w *gethWakuWrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	return w.waku.GetPrivateKey(id)
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *gethWakuWrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	return w.waku.AddKeyPair(key)
}

// DeleteKeyPair deletes the key with the specified ID if it exists.
func (w *gethWakuWrapper) DeleteKeyPair(keyID string) bool {
	return w.waku.DeleteKeyPair(keyID)
}

func (w *gethWakuWrapper) AddSymKeyDirect(key []byte) (string, error) {
	return w.waku.AddSymKeyDirect(key)
}

func (w *gethWakuWrapper) AddSymKeyFromPassword(password string) (string, error) {
	return w.waku.AddSymKeyFromPassword(password)
}

func (w *gethWakuWrapper) DeleteSymKey(id string) bool {
	return w.waku.DeleteSymKey(id)
}

func (w *gethWakuWrapper) GetSymKey(id string) ([]byte, error) {
	return w.waku.GetSymKey(id)
}

func (w *gethWakuWrapper) Subscribe(opts *types.SubscriptionOptions) (string, error) {
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

	id, err := w.waku.Subscribe(GetWakuFilterFrom(f))
	if err != nil {
		return "", err
	}

	f.(*wakuFilterWrapper).id = id
	return id, nil
}

func (w *gethWakuWrapper) GetFilter(id string) types.Filter {
	return NewWakuFilterWrapper(w.waku.GetFilter(id), id)
}

func (w *gethWakuWrapper) Unsubscribe(id string) error {
	return w.waku.Unsubscribe(id)
}

func (w *gethWakuWrapper) createFilterWrapper(id string, keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte) (types.Filter, error) {
	return NewWakuFilterWrapper(&wakucommon.Filter{
		KeyAsym:  keyAsym,
		KeySym:   keySym,
		PoW:      pow,
		AllowP2P: true,
		Topics:   topics,
		Messages: wakucommon.NewMemoryMessageStore(),
	}, id), nil
}

func (w *gethWakuWrapper) SendMessagesRequest(peerID []byte, r types.MessagesRequest) error {
	return w.waku.SendMessagesRequest(peerID, wakucommon.MessagesRequest{
		ID:     r.ID,
		From:   r.From,
		To:     r.To,
		Limit:  r.Limit,
		Cursor: r.Cursor,
		Bloom:  r.Bloom,
	})
}

// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
// which is known to implement MailServer interface, and is supposed to process this
// request and respond with a number of peer-to-peer messages (possibly expired),
// which are not supposed to be forwarded any further.
// The whisper protocol is agnostic of the format and contents of envelope.
func (w *gethWakuWrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope types.Envelope, timeout time.Duration) error {
	return w.waku.RequestHistoricMessagesWithTimeout(peerID, envelope.Unwrap().(*wakucommon.Envelope), timeout)
}

type wakuFilterWrapper struct {
	filter *wakucommon.Filter
	id     string
}

// NewWakuFilterWrapper returns an object that wraps Geth's Filter in a types interface
func NewWakuFilterWrapper(f *wakucommon.Filter, id string) types.Filter {
	if f.Messages == nil {
		panic("Messages should not be nil")
	}

	return &wakuFilterWrapper{
		filter: f,
		id:     id,
	}
}

// GetWakuFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetWakuFilterFrom(f types.Filter) *wakucommon.Filter {
	return f.(*wakuFilterWrapper).filter
}

// ID returns the filter ID
func (w *wakuFilterWrapper) ID() string {
	return w.id
}
