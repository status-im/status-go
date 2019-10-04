package gethbridge

import (
	"crypto/ecdsa"
	"time"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	whisper "github.com/status-im/whisper/whisperv6"
)

type gethWhisperWrapper struct {
	whisper *whisper.Whisper
}

// NewGethWhisperWrapper returns an object that wraps Geth's Whisper in a whispertypes interface
func NewGethWhisperWrapper(w *whisper.Whisper) whispertypes.Whisper {
	if w == nil {
		panic("w cannot be nil")
	}

	return &gethWhisperWrapper{
		whisper: w,
	}
}

// GetGethWhisperFrom retrieves the underlying whisper Whisper struct from a wrapped Whisper interface
func GetGethWhisperFrom(m whispertypes.Whisper) *whisper.Whisper {
	return m.(*gethWhisperWrapper).whisper
}

func (w *gethWhisperWrapper) PublicWhisperAPI() whispertypes.PublicWhisperAPI {
	return NewGethPublicWhisperAPIWrapper(whisper.NewPublicWhisperAPI(w.whisper))
}

func (w *gethWhisperWrapper) NewMessageStore() whispertypes.MessageStore {
	return NewGethMessageStoreWrapper(w.whisper.NewMessageStore())
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

func (w *gethWhisperWrapper) SubscribeEnvelopeEvents(eventsProxy chan<- whispertypes.EnvelopeEvent) whispertypes.Subscription {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		for e := range events {
			eventsProxy <- *NewGethEnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.whisper.SubscribeEnvelopeEvents(events))
}

// SelectedKeyPairID returns the id of currently selected key pair.
// It helps distinguish between different users w/o exposing the user identity itself.
func (w *gethWhisperWrapper) SelectedKeyPairID() string {
	return w.whisper.SelectedKeyPairID()
}

func (w *gethWhisperWrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	return w.whisper.GetPrivateKey(id)
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *gethWhisperWrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	return w.whisper.AddKeyPair(key)
}

// DeleteKeyPair deletes the specified key if it exists.
func (w *gethWhisperWrapper) DeleteKeyPair(key string) bool {
	return w.whisper.DeleteKeyPair(key)
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (w *gethWhisperWrapper) SelectKeyPair(key *ecdsa.PrivateKey) error {
	return w.whisper.SelectKeyPair(key)
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

func (w *gethWhisperWrapper) Subscribe(f whispertypes.Filter) (string, error) {
	return w.whisper.Subscribe(GetGethFilterFrom(f))
}

func (w *gethWhisperWrapper) GetFilter(id string) whispertypes.Filter {
	return NewGethFilterWrapper(w.whisper.GetFilter(id))
}

func (w *gethWhisperWrapper) Unsubscribe(id string) error {
	return w.whisper.Unsubscribe(id)
}

func (w *gethWhisperWrapper) CreateFilterWrapper(keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte, messages whispertypes.MessageStore) whispertypes.Filter {
	return NewGethFilterWrapper(&whisper.Filter{
		KeyAsym:  keyAsym,
		KeySym:   keySym,
		PoW:      pow,
		AllowP2P: true,
		Topics:   topics,
		Messages: GetGethMessageStoreFrom(messages),
	})
}

// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
// which is known to implement MailServer interface, and is supposed to process this
// request and respond with a number of peer-to-peer messages (possibly expired),
// which are not supposed to be forwarded any further.
// The whisper protocol is agnostic of the format and contents of envelope.
func (w *gethWhisperWrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope whispertypes.Envelope, timeout time.Duration) error {
	return w.whisper.RequestHistoricMessagesWithTimeout(peerID, GetGethEnvelopeFrom(envelope), timeout)
}

// SyncMessages can be sent between two Mail Servers and syncs envelopes between them.
func (w *gethWhisperWrapper) SyncMessages(peerID []byte, req whispertypes.SyncMailRequest) error {
	return w.whisper.SyncMessages(peerID, *GetGethSyncMailRequestFrom(&req))
}
