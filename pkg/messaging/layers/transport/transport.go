package transport

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/api/history"

	"github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/rfc26"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type Option func(*Transport) error

// Transport is a transport based on Whisper service.
type Transport struct {
	gocommon.PauseBroadcaster

	waku       types.Waku
	api        MessagingAPI      // backend used to send messages and read received envelopes
	privateKey *ecdsa.PrivateKey // identity of the current user
	filters    *FiltersManager
	logger     *zap.Logger
	cache      ProcessedMessageIDsCachePersistence

	envelopesMonitor *EnvelopesMonitor
	cleanFiltersFn   func() error
	quit             chan struct{}

	// received buffers decoded messages pushed from the waku adapter until
	// RetrieveRawAll drains them. It is keyed by FilterID and then by message
	// hash so that the same envelope pushed more than once before a drain is
	// stored only once — mirroring the hash-keyed per-filter store the waku
	// adapter used to keep (the persistent cache only dedups across drains).
	receivedMu sync.Mutex
	received   map[string]map[string]*types.Message

	// matchedPublisher notifies subscribers (the messenger's retrieve loop)
	// whenever a received message matched at least one listening filter.
	// Non-blocking, coalescing — the buffer above holds the actual data.
	matchedPublisher *pubsub.TypePublisher[struct{}]

	// subs owns the backend's wire subscriptions: every filter returned by a
	// FiltersManager mutation is acquired here, and every removed filter
	// released.
	subs *FilterSubscriptions
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
	filtersManager, err := NewFiltersManager(keysPersistence, privateKey, logger)
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
		privateKey:       privateKey,
		filters:          filtersManager,
		logger:           logger.With(zap.Namespace("Transport")),
		received:         make(map[string]map[string]*types.Message),
		matchedPublisher: pubsub.NewTypePublisher[struct{}](),
		subs:             newFilterSubscriptions(api, logger),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	t.cleanFiltersLoop()

	// Consume the adapter's push stream of received messages: decode, route by
	// content topic, match against our filters, and buffer for RetrieveRawAll.
	if t.api != nil {
		go t.receiveLoop()
	}

	return t, nil
}

// receiveLoop consumes the backend's envelope-event stream and turns every
// EventEnvelopeAvailable into the decode/route step. The feed is a broadcast, so
// multiple transports sharing one backend (as in tests) each see every event and
// route it against their own filters. The events channel is amply buffered so
// the backend's producer is never blocked.
func (t *Transport) receiveLoop() {
	defer gocommon.LogOnPanic()

	events := make(chan types.EnvelopeEvent, 100)
	sub := t.api.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	for {
		select {
		case <-t.quit:
			return
		case event := <-events:
			if event.Event != types.EventEnvelopeAvailable {
				continue
			}
			msg, ok := event.Data.(*types.ReceivedMessage)
			if !ok {
				continue
			}
			t.handleReceivedMessage(msg)
		}
	}
}

// handleReceivedMessage routes a received message to the listening filters on
// its (pubsub, content) topic and buffers the decoded result. A content topic
// is only a 4-byte hash of the chat id, so distinct chats can collide on it;
// a successful authenticated decrypt is what confirms which key — and hence
// which filters — the message belongs to (replacing the old per-filter
// key-hash match). Since a message decrypts with exactly one key, candidates
// are grouped by key: decoding is attempted at most once per distinct key,
// stops at the first key that succeeds, and the decoded message is fanned out
// to every filter sharing that key.
func (t *Transport) handleReceivedMessage(msg *types.ReceivedMessage) {
	if msg == nil {
		return
	}

	candidates := t.filters.WatchersByTopic(msg.PubsubTopic, msg.ContentTopic)
	if len(candidates) == 0 {
		return
	}

	type keyGroup struct {
		symKey  []byte
		privKey *ecdsa.PrivateKey
		filters []*Filter
	}
	var order []string // group iteration order, deterministic over candidates
	groups := make(map[string]*keyGroup)
	for _, filter := range candidates {
		// Filters that exist only so we can publish never receive.
		if !filter.Listen {
			continue
		}

		symKey, privKey, err := t.filterKeys(filter)
		if err != nil {
			continue
		}

		var fingerprint string
		if symKey != nil {
			fingerprint = "s:" + string(symKey)
		} else {
			fingerprint = "a:" + string(crypto.FromECDSAPub(&privKey.PublicKey))
		}

		group := groups[fingerprint]
		if group == nil {
			group = &keyGroup{symKey: symKey, privKey: privKey}
			groups[fingerprint] = group
			order = append(order, fingerprint)
		}
		group.filters = append(group.filters, filter)
	}

	var matched bool
	for _, fingerprint := range order {
		group := groups[fingerprint]

		decoded, err := t.decode(msg, group.symKey, group.privKey)
		if err != nil {
			// Wrong key (e.g. a colliding chat's message); try the next one.
			continue
		}

		for _, filter := range group.filters {
			t.bufferMessage(filter.FilterID, toMessage(decoded, filter, msg, group.privKey))
		}
		matched = true
		break // only one key can authenticate a given message
	}

	if matched {
		t.matchedPublisher.Publish(struct{}{})
	}
}

// bufferMessage stores a decoded message in the per-filter receive buffer,
// keyed by envelope hash, until RetrieveRawAll drains it.
func (t *Transport) bufferMessage(filterID string, message *types.Message) {
	t.receivedMu.Lock()
	defer t.receivedMu.Unlock()
	byHash := t.received[filterID]
	if byHash == nil {
		byHash = make(map[string]*types.Message)
		t.received[filterID] = byHash
	}
	byHash[types2.EncodeHex(message.Hash)] = message
}

// filterKeys resolves the key material used to decode messages received on a
// filter. Both symmetric and asymmetric keys come from the FiltersManager's
// runtime maps, keyed by FilterID (the same path the send side uses). Exactly
// one of the returned keys is non-nil.
func (t *Transport) filterKeys(filter *Filter) (symKey []byte, privKey *ecdsa.PrivateKey, err error) {
	if symKey, ok := t.filters.SymKey(filter.FilterID); ok {
		return symKey, nil, nil
	}
	if privKey, ok := t.filters.AsymKey(filter.FilterID); ok {
		return nil, privKey, nil
	}
	return nil, nil, errors.New("no decode key for filter")
}

// decode reverses the rfc26 payload codec using the filter's key material.
//
// The WakuMessage `version` field is deliberately ignored here, as part of the
// migration to retire it (status-im/status-go#7499): the rfc26 codec is
// independent of `version`, and the caller only reaches this point with a
// decryption key (keyless filters are skipped in handleReceivedMessage), so we
// always decrypt. status-go only ever emitted version==1, so a keyed version==0
// envelope is never genuine plaintext; once senders start advertising version==0
// (a later migration step) those messages must still decrypt here. Previously
// version==0 was passed through unencrypted — that phase-0 behaviour is removed.
func (t *Transport) decode(msg *types.ReceivedMessage, symKey []byte, privKey *ecdsa.PrivateKey) (*rfc26.DecodedPayload, error) {
	keyInfo := &rfc26.KeyInfo{}
	switch {
	case privKey != nil:
		keyInfo.Kind = rfc26.Asymmetric
		keyInfo.PrivKey = privKey
	case symKey != nil:
		keyInfo.Kind = rfc26.Symmetric
		keyInfo.SymKey = symKey
	default:
		return nil, errors.New("no decode key for filter")
	}

	return rfc26.Decode(msg.Payload, keyInfo)
}

// toMessage assembles the transport's *types.Message from a decoded payload,
// the matched filter and the raw received message.
func toMessage(decoded *rfc26.DecodedPayload, filter *Filter, raw *types.ReceivedMessage, privKey *ecdsa.PrivateKey) *types.Message {
	m := &types.Message{
		Payload:   decoded.Data,
		Padding:   decoded.Padding,
		Timestamp: uint32(raw.Timestamp / int64(time.Second)),
		Hash:      raw.Hash,
		Topic:     filter.ContentTopic,
	}
	if privKey != nil {
		m.Dst = crypto.FromECDSAPub(&privKey.PublicKey)
	}
	if decoded.PubKey != nil {
		m.Sig = crypto.FromECDSAPub(decoded.PubKey)
	}
	return m
}

func (t *Transport) InitFilters(chatIDs []FiltersToInitialize, publicKeys []*ecdsa.PublicKey) ([]*Filter, error) {
	filters, err := t.filters.Init(chatIDs, publicKeys)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filters...); err != nil {
		return nil, err
	}
	return filters, nil
}

func (t *Transport) InitPublicFilters(filtersToInit []FiltersToInitialize) ([]*Filter, error) {
	filters, err := t.filters.InitPublicFilters(filtersToInit)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filters...); err != nil {
		return nil, err
	}
	return filters, nil
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
	filters, err := t.filters.InitCommunityFilters(communityFiltersToInitialize)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filters...); err != nil {
		return nil, err
	}
	return filters, nil
}

func (t *Transport) RemoveFilters(filters []*Filter) error {
	if err := t.filters.Remove(filters...); err != nil {
		return err
	}
	t.subs.Release(filters...)
	return nil
}

func (t *Transport) RemoveFilterByChatID(chatID string) (*Filter, error) {
	filter, err := t.filters.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}
	if filter != nil {
		t.subs.Release(filter)
	}
	return filter, nil
}

func (t *Transport) ResetFilters(ctx context.Context) error {
	removed, err := t.filters.Reset()
	if err != nil {
		return err
	}
	t.subs.Release(removed...)
	return nil
}

func (t *Transport) ProcessNegotiatedSecret(secret messagingtypes.NegotiatedSecret) (*Filter, error) {
	filter, err := t.filters.LoadNegotiated(secret)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}
	return filter, nil
}

func (t *Transport) JoinPublic(chatID string) (*Filter, error) {
	filter, err := t.filters.LoadPublic(chatID, "")
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}
	return filter, nil
}

func (t *Transport) JoinPrivate(publicKey *ecdsa.PublicKey) (*Filter, error) {
	filter, err := t.filters.LoadContactCode(publicKey)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}
	return filter, nil
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
	if err := t.subs.Acquire(filters...); err != nil {
		return nil, err
	}
	return filters, nil
}

// RetrieveRawAll drains the messages pushed in by the waku adapter and decoded
// by receiveLoop, dropping any the persistent processed-message cache has
// already seen, and returns them grouped by filter.
func (t *Transport) RetrieveRawAll() (map[Filter][]*types.Message, error) {
	result := make(map[Filter][]*types.Message)
	logger := t.logger.With(zap.String("site", "retrieveRawAll"))

	t.receivedMu.Lock()
	drained := t.received
	t.received = make(map[string]map[string]*types.Message)
	t.receivedMu.Unlock()

	for filterID, byHash := range drained {
		if len(byHash) == 0 {
			continue
		}

		filter := t.filters.FilterByFilterID(filterID)
		if filter == nil {
			// Filter was removed between receipt and drain; drop its messages.
			continue
		}

		ids := make([]string, 0, len(byHash))
		for id := range byHash {
			ids = append(ids, id)
		}

		hits, err := t.cache.Hits(ids)
		if err != nil {
			logger.Error("failed to check messages exists", zap.Error(err))
			return nil, err
		}

		for _, id := range ids {
			// Exclude anything the processed-message cache has already seen.
			if !hits[id] {
				result[*filter] = append(result[*filter], byHash[id])
				logger.Debug("message not cached", zap.String("hash", id))
			} else {
				logger.Debug("message cached", zap.String("hash", id))
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

	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}

	symKey, ok := t.filters.SymKey(filter.FilterID)
	if !ok {
		return nil, errors.New("no sym key found for filter")
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

	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}

	symKey, ok := t.filters.SymKey(filter.FilterID)
	if !ok {
		return nil, errors.New("no sym key found for filter")
	}

	encoded, err := t.encode(newMessage.Payload, symKey, nil)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	filter, err := t.filters.LoadPartitioned(publicKey, t.privateKey, false)
	if err != nil {
		return nil, err
	}

	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}

	encoded, err := t.encode(newMessage.Payload, nil, publicKey)
	if err != nil {
		return nil, err
	}

	return t.api.Send(ctx, filter.PubsubTopic, filter.ContentTopic.ContentTopic(), encoded, newMessage.Ephemeral, newMessage.Priority)
}

func (t *Transport) SendPrivateOnPersonalTopic(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	filter, err := t.filters.LoadPersonal(publicKey, t.privateKey, false)
	if err != nil {
		return nil, err
	}

	if err := t.subs.Acquire(filter); err != nil {
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
	filter, err := t.filters.LoadEphemeral(&key.PublicKey, key, true)
	if err != nil {
		return nil, err
	}
	if err := t.subs.Acquire(filter); err != nil {
		return nil, err
	}
	return filter, nil
}

func (t *Transport) SendCommunityMessage(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	// We load the filter to make sure we can post on it
	filter, err := t.filters.LoadPublic(PubkeyToHex(publicKey)[2:], newMessage.PubsubTopic)
	if err != nil {
		return nil, err
	}

	if err := t.subs.Acquire(filter); err != nil {
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
	removed, err := t.filters.RemoveNoListenFilters()
	if err != nil {
		return err
	}
	t.subs.Release(removed...)
	return nil
}

// encode produces the WakuMessage version=1 payload for the given plaintext.
// Exactly one of symKey or recipient must be non-nil. The message is always
// signed with the local identity key.
func (t *Transport) encode(payload []byte, symKey []byte, recipient *ecdsa.PublicKey) ([]byte, error) {
	return rfc26.Encode(payload, symKey, recipient, t.privateKey)
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
	return len(t.waku.Peers())
}

// Peers is retained only for the Python functional tests (see tests-functional);
// it is not used by status-app.
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
	return t.matchedPublisher.Subscribe(1)
}

func (t *Transport) UnsubscribeFilterMatched(ch chan struct{}) {
	if ch == nil {
		return
	}
	t.matchedPublisher.Unsubscribe(ch)
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

func (t *Transport) IsStorenodeAvailable(peerID peer.ID) bool {
	return t.waku.IsStorenodeAvailable(peerID)
}

// Query retrieves historic messages for a single batch, selecting the store node
// internally (no peer argument). See waku.StoreClient.
func (t *Transport) Query(
	ctx context.Context,
	batch types.MailserverBatch,
	pageLimit uint64,
	shouldProcessNextPage func(int) (bool, uint64),
	processEnvelopes bool,
) error {
	return t.waku.StoreQuery(ctx, batch, pageLimit, shouldProcessNextPage, processEnvelopes)
}

func (t *Transport) SetStorenodeConfigProvider(c history.StorenodeConfigProvider) {
	t.waku.SetStorenodeConfigProvider(c)
}
