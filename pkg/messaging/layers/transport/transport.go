package transport

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/rfc26"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type transportKeysManager struct {
	waku types.Waku

	// Identity of the current user.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *transportKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in waku
	return m.waku.AddKeyPair(priv)
}

func (m *transportKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymKeyMutex.Lock()
	defer m.passToSymKeyMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.waku.AddSymKeyFromPassword(password)
	if err != nil {
		return id, err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *transportKeysManager) RawSymKey(id string) ([]byte, error) {
	return m.waku.GetSymKey(id)
}

type Option func(*Transport) error

// Transport is a transport based on Whisper service.
type Transport struct {
	gocommon.PauseBroadcaster

	waku        types.Waku
	api         MessagingAPI // backend used to send messages and read received envelopes
	keysManager *transportKeysManager
	filters     *FiltersManager
	logger      *zap.Logger
	cache       ProcessedMessageIDsCachePersistence

	envelopesMonitor *EnvelopesMonitor
	cleanFiltersFn   func() error
	quit             chan struct{}
}

// Pause signals Transport's internal goroutines to idle and cascades to
// EnvelopesMonitor and the underlying waku transport.
func (t *Transport) Pause() {
	t.MarkPaused()
	if t.envelopesMonitor != nil {
		t.envelopesMonitor.MarkPaused()
	}
	if t.waku != nil {
		if err := t.waku.Pause(); err != nil {
			t.logger.Warn("waku Pause failed", zap.Error(err))
		}
	}
}

// SetFilterBackgroundMode suppresses (background=true) or re-enables
// (background=false) filter-subscription renewal in the underlying waku
// light-client. No-op when waku is nil or in relay mode.
func (t *Transport) SetFilterBackgroundMode(background bool) {
	if t.waku != nil {
		t.waku.SetFilterBackgroundMode(background)
	}
}

// Resume signals Transport's internal goroutines to resume and cascades to
// EnvelopesMonitor and the underlying waku transport.
func (t *Transport) Resume() {
	t.MarkResumed()
	if t.envelopesMonitor != nil {
		t.envelopesMonitor.MarkResumed()
	}
	if t.waku != nil {
		if err := t.waku.Resume(); err != nil {
			t.logger.Warn("waku Resume failed", zap.Error(err))
		}
	}
}

var cleanFiltersLoopInterval = 5 * time.Minute

// NewTransport returns a new Transport.
// TODO: leaving a chat should verify that for a given public key
//
//	there are no other chats. It may happen that we leave a private chat
//	but still have a public chat for a given public key.
func NewTransport(
	waku types.Waku,
	privateKey *ecdsa.PrivateKey,
	keysPersistence KeysPersistence,
	messageIDsCachePersistence ProcessedMessageIDsCachePersistence,
	envelopesMonitorConfig *EnvelopesMonitorConfig,
	logger *zap.Logger,
	opts ...Option,
) (*Transport, error) {
	filtersManager, err := NewFiltersManager(keysPersistence, waku, privateKey, logger)
	if err != nil {
		return nil, err
	}

	var envelopesMonitor *EnvelopesMonitor
	if envelopesMonitorConfig != nil {
		envelopesMonitor = NewEnvelopesMonitor(waku, *envelopesMonitorConfig)
		envelopesMonitor.Start()
	}

	var api MessagingAPI
	if waku != nil {
		api = waku
	}
	t := &Transport{
		waku:             waku,
		api:              api,
		cache:            messageIDsCachePersistence,
		envelopesMonitor: envelopesMonitor,
		quit:             make(chan struct{}),
		keysManager: &transportKeysManager{
			waku:              waku,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
		filters: filtersManager,
		logger:  logger.With(zap.Namespace("Transport")),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	t.cleanFiltersLoop()

	return t, nil
}

func (t *Transport) InitFilters(chatIDs []FiltersToInitialize, publicKeys []*ecdsa.PublicKey) ([]*Filter, error) {
	return t.filters.Init(chatIDs, publicKeys)
}

func (t *Transport) InitPublicFilters(filtersToInit []FiltersToInitialize) ([]*Filter, error) {
	return t.filters.InitPublicFilters(filtersToInit)
}

func (t *Transport) Filters() []*Filter {
	return t.filters.Filters()
}

func (t *Transport) FilterByChatID(chatID string) *Filter {
	return t.filters.FilterByChatID(chatID)
}

func (t *Transport) FilterByTopic(topic []byte) *Filter {
	return t.filters.FilterByTopic(topic)
}

func (t *Transport) FiltersByIdentities(identities []string) []*Filter {
	return t.filters.FiltersByIdentities(identities)
}

func (t *Transport) InitCommunityFilters(communityFiltersToInitialize []CommunityFilterToInitialize) ([]*Filter, error) {
	return t.filters.InitCommunityFilters(communityFiltersToInitialize)
}

func (t *Transport) RemoveFilters(filters []*Filter) error {
	return t.filters.Remove(context.Background(), filters...)
}

func (t *Transport) RemoveFilterByChatID(chatID string) (*Filter, error) {
	return t.filters.RemoveFilterByChatID(chatID)
}

func (t *Transport) ResetFilters(ctx context.Context) error {
	return t.filters.Reset(ctx)
}

func (t *Transport) ProcessNegotiatedSecret(secret messagingtypes.NegotiatedSecret) (*Filter, error) {
	filter, err := t.filters.LoadNegotiated(secret)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (t *Transport) JoinPublic(chatID string) (*Filter, error) {
	return t.filters.LoadPublic(chatID, "")
}

func (t *Transport) JoinPrivate(publicKey *ecdsa.PublicKey) (*Filter, error) {
	return t.filters.LoadContactCode(publicKey)
}

func (t *Transport) JoinGroup(publicKeys []*ecdsa.PublicKey) ([]*Filter, error) {
	var filters []*Filter
	for _, pk := range publicKeys {
		f, err := t.filters.LoadContactCode(pk)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)

	}
	return filters, nil
}

func (t *Transport) GetStats() types.StatsSummary {
	return t.waku.GetStats()
}

func (t *Transport) RetrieveRawAll() (map[Filter][]*types.Message, error) {
	result := make(map[Filter][]*types.Message)
	logger := t.logger.With(zap.String("site", "retrieveRawAll"))

	for _, filter := range t.filters.Filters() {
		msgs, err := t.api.GetFilterMessages(filter.FilterID)
		if err != nil {
			logger.Warn("failed to fetch messages", zap.Error(err))
			continue
		}
		// Don't pull from filters we don't listen to
		if !filter.Listen {
			for _, msg := range msgs {
				t.waku.MarkP2PMessageAsProcessed(common.BytesToHash(msg.Hash))
			}
			continue
		}

		if len(msgs) == 0 {
			continue
		}

		ids := make([]string, len(msgs))
		for i := range msgs {
			id := types2.EncodeHex(msgs[i].Hash)
			ids[i] = id
		}

		hits, err := t.cache.Hits(ids)
		if err != nil {
			logger.Error("failed to check messages exists", zap.Error(err))
			return nil, err
		}

		for i := range msgs {
			// Exclude anything that is a cache hit
			if !hits[types2.EncodeHex(msgs[i].Hash)] {
				result[*filter] = append(result[*filter], msgs[i])
				logger.Debug("message not cached", zap.String("hash", types2.EncodeHex(msgs[i].Hash)))
			} else {
				logger.Debug("message cached", zap.String("hash", types2.EncodeHex(msgs[i].Hash)))
				t.waku.MarkP2PMessageAsProcessed(common.BytesToHash(msgs[i].Hash))
			}
		}

	}

	return result, nil
}

// SendPublic sends a new message using the Whisper service.
// For public filters, chat name is used as an ID as well as
// a topic.
func (t *Transport) SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error) {
	filter, err := t.filters.LoadPublic(chatName, newMessage.PubsubTopic)
	if err != nil {
		return nil, err
	}

	symKey, err := t.keysManager.RawSymKey(filter.SymKeyID)
	if err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, symKey, nil)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	filter, err := t.filters.LoadNegotiated(messagingtypes.NegotiatedSecret{
		PublicKey: publicKey,
		Key:       secret,
	})
	if err != nil {
		return nil, err
	}

	symKey, err := t.keysManager.RawSymKey(filter.SymKeyID)
	if err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, symKey, nil)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	filter, err := t.filters.LoadPartitioned(publicKey, t.keysManager.privateKey, false)
	if err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, nil, publicKey)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) SendPrivateOnPersonalTopic(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	filter, err := t.filters.LoadPersonal(publicKey, t.keysManager.privateKey, false)
	if err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, nil, publicKey)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) PersonalTopicFilter() *Filter {
	return t.filters.PersonalTopicFilter()
}

func (t *Transport) LoadKeyFilters(key *ecdsa.PrivateKey) (*Filter, error) {
	return t.filters.LoadEphemeral(&key.PublicKey, key, true)
}

func (t *Transport) SendCommunityMessage(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	// We load the filter to make sure we can post on it
	filter, err := t.filters.LoadPublic(PubkeyToHex(publicKey)[2:], newMessage.PubsubTopic)
	if err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, nil, publicKey)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) cleanFilters() error {
	if t.cleanFiltersFn != nil {
		return t.cleanFiltersFn()
	}
	if t.filters == nil {
		return nil
	}
	return t.filters.RemoveNoListenFilters()
}

// encode produces the WakuMessage version=1 payload for the given plaintext.
// Exactly one of symKey or recipient must be non-nil. The message is always
// signed with the local identity key.
func (t *Transport) encode(payload []byte, symKey []byte, recipient *ecdsa.PublicKey) ([]byte, error) {
	return rfc26.Encode(payload, symKey, recipient, t.keysManager.privateKey)
}

// Track records sent envelopes for delivery monitoring. It is the
// single-identifier convenience wrapper around TrackMany.
func (t *Transport) Track(identifier []byte, hashes [][]byte, newMessages []*types.NewMessage) {
	t.TrackMany([][]byte{identifier}, hashes, newMessages)
}

func (t *Transport) TrackMany(identifiers [][]byte, hashes [][]byte, newMessages []*types.NewMessage) {
	if t.envelopesMonitor == nil {
		return
	}

	envelopeHashes := make([]types2.Hash, len(hashes))
	for i, hash := range hashes {
		envelopeHashes[i] = types2.BytesToHash(hash)
	}

	err := t.envelopesMonitor.Add(identifiers, envelopeHashes, newMessages)
	if err != nil {
		t.logger.Error("failed to track messages", zap.Error(err))
	}
}

// GetCurrentTime returns the current unix timestamp in milliseconds
func (t *Transport) GetCurrentTime() uint64 {
	return t.waku.GetCurrentTime()
}

func (t *Transport) MaxMessageSize() uint32 {
	return t.waku.MaxMessageSize()
}

func (t *Transport) Stop() error {
	close(t.quit)

	if t.envelopesMonitor != nil {
		t.envelopesMonitor.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := t.ResetFilters(ctx)
	if err != nil {
		return err
	}

	return nil
}

// cleanFiltersLoop cleans up the topic we create for the only purpose
// of sending messages.
// Whenever we send a message we also need to listen to that particular topic
// but in case of asymettric topics, we are not interested in listening to them.
// We therefore periodically clean them up so we don't receive unnecessary data.

func (t *Transport) cleanFiltersLoop() {
	go func() {
		defer gocommon.LogOnPanic()
		sub := t.Subscribe()
		defer sub.Unsubscribe()
		pt := gocommon.NewPausableTicker(gocommon.PausableTickerConfig{
			Interval: cleanFiltersLoopInterval,
			OnTick: func() {
				err := t.cleanFilters()
				if err != nil {
					t.logger.Error("failed to clean up topics", zap.Error(err))
				}
			},
		}, sub.C())
		pt.Run(t.quit)
	}()
}

func (t *Transport) PeerCount() int {
	return t.waku.PeerCount()
}

func (t *Transport) Peers() types.PeerStats {
	return t.waku.Peers()
}

// ConfirmMessagesProcessed marks the messages as processed in the cache so
// they won't be passed to the next layer anymore
func (t *Transport) ConfirmMessagesProcessed(ids []string, timestamp uint64) error {
	t.logger.Debug("confirming message processed", zap.Any("ids", ids), zap.Any("timestamp", timestamp))
	return t.cache.Add(ids, timestamp)
}

// CleanMessagesProcessed clears the messages that are older than timestamp
func (t *Transport) CleanMessagesProcessed(timestamp uint64) error {
	return t.cache.Clean(timestamp)
}

func (t *Transport) SetEnvelopeEventsHandler(handler EnvelopeEventsHandler) error {
	if t.envelopesMonitor == nil {
		return errors.New("Current transport has no envelopes monitor")
	}
	t.envelopesMonitor.handler = handler
	return nil
}

func (t *Transport) ClearProcessedMessageIDsCache() error {
	t.logger.Debug("clearing processed messages cache")
	t.waku.ClearEnvelopesCache()
	return t.cache.Clear()
}

func PubkeyToHex(key *ecdsa.PublicKey) string {
	return types2.EncodeHex(crypto.FromECDSAPub(key))
}

func (t *Transport) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	return t.waku.ListenAddresses()
}

func (t *Transport) ENR() (*enode.Node, error) {
	return t.waku.ENR()
}

func (t *Transport) AddRelayPeer(address multiaddr.Multiaddr) (peer.ID, error) {
	return t.waku.AddRelayPeer(address)
}

func (t *Transport) DialPeer(address multiaddr.Multiaddr) error {
	return t.waku.DialPeer(address)
}

func (t *Transport) DialPeerByID(peerID peer.ID) error {
	return t.waku.DialPeerByID(peerID)
}

func (t *Transport) DropPeer(peerID peer.ID) error {
	return t.waku.DropPeer(peerID)
}

func (t *Transport) MarkP2PMessageAsProcessed(hash common.Hash) {
	t.waku.MarkP2PMessageAsProcessed(hash)
}

func (t *Transport) ConnectionChanged(state connection.State) {
	t.waku.ConnectionChanged(state)
}

// Subscribe to a pubsub topic
func (t *Transport) SubscribeToPubsubTopic(topic string) error {
	t.logger.Debug("subscribing to pubsub topic", zap.String("pubsubtopic", topic))
	return t.waku.SubscribeToPubsubTopic(topic)
}

func (t *Transport) ConfirmMessageDelivered(messageID string) {
	if t.envelopesMonitor == nil {
		return
	}
	hashes, ok := t.envelopesMonitor.messageEnvelopeHashes[messageID]
	if !ok {
		return
	}
	commHashes := make([]common.Hash, len(hashes))
	for i, h := range hashes {
		commHashes[i] = common.BytesToHash(h[:])
	}
	t.waku.ConfirmMessageDelivered(commHashes)
}

func (t *Transport) SetCriteriaForMissingMessageVerification(peerInfo peer.AddrInfo, filters []*Filter) {
	topicMap := make(map[string]map[types.TopicType]struct{})
	for _, f := range filters {
		if !f.Listen || f.Ephemeral {
			continue
		}

		_, ok := topicMap[f.PubsubTopic]
		if !ok {
			topicMap[f.PubsubTopic] = make(map[types.TopicType]struct{})
		}

		topicMap[f.PubsubTopic][f.ContentTopic] = struct{}{}
	}

	for pubsubTopic, contentTopics := range topicMap {
		ctList := maps.Keys(contentTopics)
		err := t.waku.SetCriteriaForMissingMessageVerification(peerInfo, pubsubTopic, ctList)
		if err != nil {
			t.logger.Error("could not check for missing messages",
				zap.Error(err),
				zap.Stringer("peerID", peerInfo.ID),
				zap.String("pubsubTopic", pubsubTopic),
				zap.Stringers("contentTopics", ctList))
			return
		}
	}
}

func (t *Transport) GetActiveStorenode() peer.AddrInfo {
	return t.waku.GetActiveStorenode()
}

func (t *Transport) DisconnectActiveStorenode(ctx context.Context, backoffReason time.Duration, shouldCycle bool) {
	t.waku.DisconnectActiveStorenode(ctx, backoffReason, shouldCycle)
}

// SubscribeFilterMatched returns a channel notified (non-blocking, coalescing) whenever
// an incoming envelope matches at least one installed filter. Callers must call
// UnsubscribeFilterMatched with the returned channel when done.
func (t *Transport) SubscribeFilterMatched() chan struct{} {
	if t.waku == nil {
		return nil
	}
	return t.waku.SubscribeFilterMatched()
}

func (t *Transport) UnsubscribeFilterMatched(ch chan struct{}) {
	if t.waku == nil || ch == nil {
		return
	}
	t.waku.UnsubscribeFilterMatched(ch)
}

func (t *Transport) OnStorenodeChanged() <-chan peer.ID {
	return t.waku.OnStorenodeChanged()
}

func (t *Transport) OnStorenodeNotWorking() <-chan struct{} {
	return t.waku.OnStorenodeNotWorking()
}

func (t *Transport) OnStorenodeAvailable() <-chan peer.ID {
	return t.waku.OnStorenodeAvailable()
}

func (t *Transport) WaitForAvailableStoreNode(ctx context.Context) bool {
	return t.waku.WaitForAvailableStoreNode(ctx)
}

func (t *Transport) IsStorenodeAvailable(peerID peer.ID) bool {
	return t.waku.IsStorenodeAvailable(peerID)
}

func (t *Transport) PerformStorenodeTask(fn func() error, opts ...history.StorenodeTaskOption) error {
	return t.waku.PerformStorenodeTask(fn, opts...)
}

func (t *Transport) ProcessMailserverBatch(
	ctx context.Context,
	batch types.MailserverBatch,
	storenode peer.AddrInfo,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return t.waku.ProcessMailserverBatch(ctx, batch, storenode, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (t *Transport) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	t.waku.SetStorenodeConfigProvider(c)
}
