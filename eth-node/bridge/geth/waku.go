package gethbridge

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
)

type GethWakuWrapper struct {
	waku *waku.Waku
}

// NewGethWakuWrapper returns an object that wraps Geth's Waku in a types interface
func NewGethWakuWrapper(w *waku.Waku) types.Waku {
	if w == nil {
		panic("waku cannot be nil")
	}

	return &GethWakuWrapper{
		waku: w,
	}
}

// GetGethWhisperFrom retrieves the underlying whisper Whisper struct from a wrapped Whisper interface
func GetGethWakuFrom(m types.Waku) *waku.Waku {
	return m.(*GethWakuWrapper).waku
}

func (w *GethWakuWrapper) PublicWakuAPI() types.PublicWakuAPI {
	return NewGethPublicWakuAPIWrapper(waku.NewPublicWakuAPI(w.waku))
}

func (w *GethWakuWrapper) Version() uint {
	return 1
}

// Added for compatibility with waku V2
func (w *GethWakuWrapper) PeerCount() int {
	return -1
}

// Added for compatibility with waku V2
func (w *GethWakuWrapper) StartDiscV5() error {
	return errors.New("not available in WakuV1")
}

// Added for compatibility with waku V2
func (w *GethWakuWrapper) StopDiscV5() error {
	return errors.New("not available in WakuV1")
}

// PeerCount function only added for compatibility with waku V2
func (w *GethWakuWrapper) AddStorePeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return "", errors.New("not available in WakuV1")
}

// SubscribeToPubsubTopic function only added for compatibility with waku V2
func (w *GethWakuWrapper) SubscribeToPubsubTopic(topic string, optPublicKey *ecdsa.PublicKey) error {
	// not available in WakuV1
	return errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) UnsubscribeFromPubsubTopic(topic string) error {
	// not available in WakuV1
	return errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) RetrievePubsubTopicKey(topic string) (*ecdsa.PrivateKey, error) {
	// not available in WakuV1
	return nil, errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) StorePubsubTopicKey(topic string, privKey *ecdsa.PrivateKey) error {
	// not available in WakuV1
	return errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) RemovePubsubTopicKey(topic string) error {
	// not available in WakuV1
	return errors.New("not available in WakuV1")
}

// AddRelayPeer function only added for compatibility with waku V2
func (w *GethWakuWrapper) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return "", errors.New("not available in WakuV1")
}

// DialPeer function only added for compatibility with waku V2
func (w *GethWakuWrapper) DialPeer(address multiaddr.Multiaddr) error {
	return errors.New("not available in WakuV1")
}

// DialPeerByID function only added for compatibility with waku V2
func (w *GethWakuWrapper) DialPeerByID(peerID peer.ID) error {
	return errors.New("not available in WakuV1")
}

// ListenAddresses function only added for compatibility with waku V2
func (w *GethWakuWrapper) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return nil, errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) RelayPeersByTopic(topic string) (*types.PeerList, error) {
	return nil, errors.New("not available in WakuV1")
}

// ENR function only added for compatibility with waku V2
func (w *GethWakuWrapper) ENR() (*enode.Node, error) {
	return nil, errors.New("not available in WakuV1")
}

// PeerCount function only added for compatibility with waku V2
func (w *GethWakuWrapper) DropPeer(peerID peer.ID) error {
	return errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) SubscribeToConnStatusChanges() (*types.ConnStatusSubscription, error) {
	return nil, errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) SetCriteriaForMissingMessageVerification(peerID peer.ID, pubsubTopic string, contentTopics []types.TopicType) error {
	return errors.New("not available in WakuV1")
}

// Peers function only added for compatibility with waku V2
func (w *GethWakuWrapper) Peers() types.PeerStats {
	p := make(types.PeerStats)
	return p
}

// MinPow returns the PoW value required by this node.
func (w *GethWakuWrapper) MinPow() float64 {
	return w.waku.MinPow()
}

// MaxMessageSize returns the MaxMessageSize set
func (w *GethWakuWrapper) MaxMessageSize() uint32 {
	return w.waku.MaxMessageSize()
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (w *GethWakuWrapper) BloomFilter() []byte {
	return w.waku.BloomFilter()
}

// GetCurrentTime returns current time.
func (w *GethWakuWrapper) GetCurrentTime() time.Time {
	return w.waku.CurrentTime()
}

func (w *GethWakuWrapper) SubscribeEnvelopeEvents(eventsProxy chan<- types.EnvelopeEvent) types.Subscription {
	events := make(chan wakucommon.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	go func() {
		defer gocommon.LogOnPanic()
		for e := range events {
			eventsProxy <- *NewWakuEnvelopeEventWrapper(&e)
		}
	}()

	return NewGethSubscriptionWrapper(w.waku.SubscribeEnvelopeEvents(events))
}

func (w *GethWakuWrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	return w.waku.GetPrivateKey(id)
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *GethWakuWrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	return w.waku.AddKeyPair(key)
}

// DeleteKeyPair deletes the key with the specified ID if it exists.
func (w *GethWakuWrapper) DeleteKeyPair(keyID string) bool {
	return w.waku.DeleteKeyPair(keyID)
}

func (w *GethWakuWrapper) AddSymKeyDirect(key []byte) (string, error) {
	return w.waku.AddSymKeyDirect(key)
}

func (w *GethWakuWrapper) AddSymKeyFromPassword(password string) (string, error) {
	return w.waku.AddSymKeyFromPassword(password)
}

func (w *GethWakuWrapper) DeleteSymKey(id string) bool {
	return w.waku.DeleteSymKey(id)
}

func (w *GethWakuWrapper) GetSymKey(id string) ([]byte, error) {
	return w.waku.GetSymKey(id)
}

func (w *GethWakuWrapper) Subscribe(opts *types.SubscriptionOptions) (string, error) {
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

func (w *GethWakuWrapper) GetStats() types.StatsSummary {
	return w.waku.GetStats()
}

func (w *GethWakuWrapper) GetFilter(id string) types.Filter {
	return NewWakuFilterWrapper(w.waku.GetFilter(id), id)
}

func (w *GethWakuWrapper) Unsubscribe(ctx context.Context, id string) error {
	return w.waku.Unsubscribe(id)
}

func (w *GethWakuWrapper) UnsubscribeMany(ids []string) error {
	return w.waku.UnsubscribeMany(ids)
}

func (w *GethWakuWrapper) createFilterWrapper(id string, keyAsym *ecdsa.PrivateKey, keySym []byte, pow float64, topics [][]byte) (types.Filter, error) {
	return NewWakuFilterWrapper(&wakucommon.Filter{
		KeyAsym:  keyAsym,
		KeySym:   keySym,
		PoW:      pow,
		AllowP2P: true,
		Topics:   topics,
		Messages: wakucommon.NewMemoryMessageStore(),
	}, id), nil
}

func (w *GethWakuWrapper) ProcessingP2PMessages() bool {
	return w.waku.ProcessingP2PMessages()
}

func (w *GethWakuWrapper) MarkP2PMessageAsProcessed(hash common.Hash) {
	w.waku.MarkP2PMessageAsProcessed(hash)
}

func (w *GethWakuWrapper) ConnectionChanged(_ connection.State) {}

func (w *GethWakuWrapper) ClearEnvelopesCache() {
	w.waku.ClearEnvelopesCache()
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

func (w *GethWakuWrapper) ConfirmMessageDelivered(hashes []common.Hash) {
}

func (w *GethWakuWrapper) PeerID() peer.ID {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) GetActiveStorenode() peer.ID {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) OnStorenodeChanged() <-chan peer.ID {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) OnStorenodeNotWorking() <-chan struct{} {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) OnStorenodeAvailable() <-chan peer.ID {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) WaitForAvailableStoreNode(ctx context.Context) bool {
	return false
}

func (w *GethWakuWrapper) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	panic("not available in WakuV1")
}

func (w *GethWakuWrapper) ProcessMailserverBatch(
	ctx context.Context,
	batch types.MailserverBatch,
	storenodeID peer.ID,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return errors.New("not available in WakuV1")
}

func (w *GethWakuWrapper) IsStorenodeAvailable(peerID peer.ID) bool {
	panic("not available in WakuV1")

}

func (w *GethWakuWrapper) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	panic("not available in WakuV1")

}

func (w *GethWakuWrapper) DisconnectActiveStorenode(ctx context.Context, backoff time.Duration, shouldCycle bool) {
	panic("not available in WakuV1")
}
