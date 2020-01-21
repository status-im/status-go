package gethbridge

import (
	"crypto/ecdsa"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/whisper/v6"
)

type gethWhisperWrapper struct {
	whisper *whisper.Whisper
}

// NewGethWhisperWrapper returns an object that wraps Geth's Whisper in a types interface
func NewGethWhisperWrapper(w *whisper.Whisper) types.Whisper {
	if w == nil {
		panic("w cannot be nil")
	}

	return &gethWhisperWrapper{
		whisper: w,
	}
}

// GetGethWhisperFrom retrieves the underlying whisper Whisper struct from a wrapped Whisper interface
func GetGethWhisperFrom(m types.Whisper) *whisper.Whisper {
	return m.(*gethWhisperWrapper).whisper
}

func (w *gethWhisperWrapper) PublicWhisperAPI() types.PublicWhisperAPI {
	return NewGethPublicWhisperAPIWrapper(whisper.NewPublicWhisperAPI(w.whisper))
}

// MinPow returns the PoW value required by this node.
func (w *gethWhisperWrapper) MinPow() float64 {
	return w.whisper.MinPow()
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (w *gethWhisperWrapper) BloomFilter() []byte {
	return w.whisper.BloomFilter()
}

// GetCurrentTime returns current time.
func (w *gethWhisperWrapper) GetCurrentTime() time.Time {
	return w.whisper.GetCurrentTime()
}

// SetTimeSource assigns a particular source of time to a whisper object.
func (w *gethWhisperWrapper) SetTimeSource(timesource func() time.Time) {
	w.whisper.SetTimeSource(timesource)
}

func (w *gethWhisperWrapper) SubscribeEnvelopeEvents(eventsProxy chan<- types.EnvelopeEvent) types.Subscription {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		for e := range events {
			eventsProxy <- *NewWhisperEnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.whisper.SubscribeEnvelopeEvents(events))
}

func (w *gethWhisperWrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	return w.whisper.GetPrivateKey(id)
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *gethWhisperWrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	return w.whisper.AddKeyPair(key)
}

// DeleteKeyPair deletes the key with the specified ID if it exists.
func (w *gethWhisperWrapper) DeleteKeyPair(keyID string) bool {
	return w.whisper.DeleteKeyPair(keyID)
}

// DeleteKeyPairs removes all cryptographic identities known to the node
func (w *gethWhisperWrapper) DeleteKeyPairs() error {
	return w.whisper.DeleteKeyPairs()
}

func (w *gethWhisperWrapper) AddSymKeyDirect(key []byte) (string, error) {
	return w.whisper.AddSymKeyDirect(key)
}

func (w *gethWhisperWrapper) AddSymKeyFromPassword(password string) (string, error) {
	return w.whisper.AddSymKeyFromPassword(password)
}

func (w *gethWhisperWrapper) DeleteSymKey(id string) bool {
	return w.whisper.DeleteSymKey(id)
}

func (w *gethWhisperWrapper) GetSymKey(id string) ([]byte, error) {
	return w.whisper.GetSymKey(id)
}

func (w *gethWhisperWrapper) Subscribe(opts *types.SubscriptionOptions) (string, error) {
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

	id, err := w.whisper.Subscribe(GetWhisperFilterFrom(f))
	if err != nil {
		return "", err
	}

	f.(*whisperFilterWrapper).id = id
	return id, nil
}

func (w *gethWhisperWrapper) GetFilter(id string) types.Filter {
	return NewWhisperFilterWrapper(w.whisper.GetFilter(id), id)
}

func (w *gethWhisperWrapper) Unsubscribe(id string) error {
	return w.whisper.Unsubscribe(id)
}

func (w *gethWhisperWrapper) createFilterWrapper(id string, keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte) (types.Filter, error) {
	return NewWhisperFilterWrapper(&whisper.Filter{
		KeyAsym:  keyAsym,
		KeySym:   keySym,
		PoW:      pow,
		AllowP2P: true,
		Topics:   topics,
		Messages: whisper.NewMemoryMessageStore(),
	}, id), nil
}

func (w *gethWhisperWrapper) SendMessagesRequest(peerID []byte, r types.MessagesRequest) error {
	return w.whisper.SendMessagesRequest(peerID, whisper.MessagesRequest{
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
func (w *gethWhisperWrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope types.Envelope, timeout time.Duration) error {
	return w.whisper.RequestHistoricMessagesWithTimeout(peerID, envelope.Unwrap().(*whisper.Envelope), timeout)
}

// SyncMessages can be sent between two Mail Servers and syncs envelopes between them.
func (w *gethWhisperWrapper) SyncMessages(peerID []byte, req types.SyncMailRequest) error {
	return w.whisper.SyncMessages(peerID, *GetGethSyncMailRequestFrom(&req))
}

type whisperFilterWrapper struct {
	filter *whisper.Filter
	id     string
}

// NewWhisperFilterWrapper returns an object that wraps Geth's Filter in a types interface
func NewWhisperFilterWrapper(f *whisper.Filter, id string) types.Filter {
	if f.Messages == nil {
		panic("Messages should not be nil")
	}

	return &whisperFilterWrapper{
		filter: f,
		id:     id,
	}
}

// GetWhisperFilterFrom retrieves the underlying whisper Filter struct from a wrapped Filter interface
func GetWhisperFilterFrom(f types.Filter) *whisper.Filter {
	return f.(*whisperFilterWrapper).filter
}

// ID returns the filter ID
func (w *whisperFilterWrapper) ID() string {
	return w.id
}
