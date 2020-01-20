package waku

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type wakuServiceKeysManager struct {
	waku types.Waku

	// Identity of the current user.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *wakuServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in waku
	return m.waku.AddKeyPair(priv)
}

func (m *wakuServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
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

func (m *wakuServiceKeysManager) RawSymKey(id string) ([]byte, error) {
	return m.waku.GetSymKey(id)
}

type Option func(*WakuServiceTransport) error

// WakuServiceTransport is a transport based on Whisper service.
type WakuServiceTransport struct {
	waku        types.Waku
	api         types.PublicWakuAPI // only PublicWakuAPI implements logic to send messages
	keysManager *wakuServiceKeysManager
	filters     *transport.FiltersManager
	logger      *zap.Logger

	mailservers      []string
	envelopesMonitor *EnvelopesMonitor
}

// NewWakuServiceTransport returns a new WakuServiceTransport.
// TODO: leaving a chat should verify that for a given public key
//       there are no other chats. It may happen that we leave a private chat
//       but still have a public chat for a given public key.
func NewWakuServiceTransport(
	waku types.Waku,
	privateKey *ecdsa.PrivateKey,
	db *sql.DB,
	mailservers []string,
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig,
	logger *zap.Logger,
	opts ...Option,
) (*WakuServiceTransport, error) {
	filtersManager, err := transport.NewFiltersManager(newSQLitePersistence(db), waku, privateKey, logger)
	if err != nil {
		return nil, err
	}

	var envelopesMonitor *EnvelopesMonitor
	if envelopesMonitorConfig != nil {
		envelopesMonitor = NewEnvelopesMonitor(waku, *envelopesMonitorConfig)
		envelopesMonitor.Start()
	}

	var api types.PublicWhisperAPI
	if waku != nil {
		api = waku.PublicWakuAPI()
	}
	t := &WakuServiceTransport{
		waku:             waku,
		api:              api,
		envelopesMonitor: envelopesMonitor,
		keysManager: &wakuServiceKeysManager{
			waku:              waku,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
		filters:     filtersManager,
		mailservers: mailservers,
		logger:      logger.With(zap.Namespace("WakuServiceTransport")),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (a *WakuServiceTransport) InitFilters(chatIDs []string, publicKeys []*ecdsa.PublicKey) ([]*transport.Filter, error) {
	return a.filters.Init(chatIDs, publicKeys)
}

func (a *WakuServiceTransport) Filters() []*transport.Filter {
	return a.filters.Filters()
}

// DEPRECATED
func (a *WakuServiceTransport) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return a.filters.InitWithFilters(filters)
}

// DEPRECATED
func (a *WakuServiceTransport) RemoveFilters(filters []*transport.Filter) error {
	return a.filters.Remove(filters...)
}

func (a *WakuServiceTransport) ResetFilters() error {
	return a.filters.Reset()
}

func (a *WakuServiceTransport) ProcessNegotiatedSecret(secret types.NegotiatedSecret) (*transport.Filter, error) {
	filter, err := a.filters.LoadNegotiated(secret)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (a *WakuServiceTransport) JoinPublic(chatID string) error {
	_, err := a.filters.LoadPublic(chatID)
	return err
}

func (a *WakuServiceTransport) LeavePublic(chatID string) error {
	chat := a.filters.Filter(chatID)
	if chat != nil {
		return nil
	}
	return a.filters.Remove(chat)
}

func (a *WakuServiceTransport) JoinPrivate(publicKey *ecdsa.PublicKey) error {
	_, err := a.filters.LoadDiscovery()
	if err != nil {
		return err
	}
	_, err = a.filters.LoadContactCode(publicKey)
	return err
}

func (a *WakuServiceTransport) LeavePrivate(publicKey *ecdsa.PublicKey) error {
	filters := a.filters.FiltersByPublicKey(publicKey)
	return a.filters.Remove(filters...)
}

func (a *WakuServiceTransport) JoinGroup(publicKeys []*ecdsa.PublicKey) error {
	_, err := a.filters.LoadDiscovery()
	if err != nil {
		return err
	}
	for _, pk := range publicKeys {
		_, err = a.filters.LoadContactCode(pk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *WakuServiceTransport) LeaveGroup(publicKeys []*ecdsa.PublicKey) error {
	for _, publicKey := range publicKeys {
		filters := a.filters.FiltersByPublicKey(publicKey)
		if err := a.filters.Remove(filters...); err != nil {
			return err
		}
	}
	return nil
}

type Message struct {
	Message *types.Message
	Public  bool
}

func (a *WakuServiceTransport) RetrieveAllMessages() ([]Message, error) {
	var messages []Message

	for _, filter := range a.filters.Filters() {
		filterMsgs, err := a.api.GetFilterMessages(filter.FilterID)
		if err != nil {
			return nil, err
		}

		for _, m := range filterMsgs {
			messages = append(messages, Message{
				Message: m,
				Public:  filter.IsPublic(),
			})
		}
	}

	return messages, nil
}

func (a *WakuServiceTransport) RetrievePublicMessages(chatID string) ([]*types.Message, error) {
	filter, err := a.filters.LoadPublic(chatID)
	if err != nil {
		return nil, err
	}

	return a.api.GetFilterMessages(filter.FilterID)
}

func (a *WakuServiceTransport) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*types.Message, error) {
	chats := a.filters.FiltersByPublicKey(publicKey)
	discoveryChats, err := a.filters.Init(nil, nil)
	if err != nil {
		return nil, err
	}

	var result []*types.Message

	for _, chat := range append(chats, discoveryChats...) {
		filterMsgs, err := a.api.GetFilterMessages(chat.FilterID)
		if err != nil {
			return nil, err
		}

		result = append(result, filterMsgs...)
	}

	return result, nil
}

func (a *WakuServiceTransport) RetrieveRawAll() (map[transport.Filter][]*types.Message, error) {
	result := make(map[transport.Filter][]*types.Message)

	allFilters := a.filters.Filters()
	for _, filter := range allFilters {
		msgs, err := a.api.GetFilterMessages(filter.FilterID)
		if err != nil {
			continue
		}
		result[*filter] = append(result[*filter], msgs...)
	}

	return result, nil
}

// SendPublic sends a new message using the Whisper service.
// For public filters, chat name is used as an ID as well as
// a topic.
func (a *WakuServiceTransport) SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadPublic(chatName)
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = filter.SymKeyID
	newMessage.Topic = filter.Topic

	return a.api.Post(ctx, *newMessage)
}

func (a *WakuServiceTransport) SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadNegotiated(types.NegotiatedSecret{
		PublicKey: publicKey,
		Key:       secret,
	})
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = filter.SymKeyID
	newMessage.Topic = filter.Topic
	newMessage.PublicKey = nil

	return a.api.Post(ctx, *newMessage)
}

func (a *WakuServiceTransport) SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadPartitioned(publicKey)
	if err != nil {
		return nil, err
	}

	newMessage.Topic = filter.Topic
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.api.Post(ctx, *newMessage)
}

func (a *WakuServiceTransport) SendPrivateOnDiscovery(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	// There is no need to load any chat
	// because listening on the discovery topic
	// is done automatically.
	// TODO: change this anyway, it should be explicit
	// and idempotent.

	newMessage.Topic = types.BytesToTopic(transport.ToTopic(transport.DiscoveryTopic()))
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.api.Post(ctx, *newMessage)
}

func (a *WakuServiceTransport) addSig(newMessage *types.NewMessage) error {
	sigID, err := a.keysManager.AddOrGetKeyPair(a.keysManager.privateKey)
	if err != nil {
		return err
	}
	newMessage.SigID = sigID
	return nil
}

func (a *WakuServiceTransport) Track(identifiers [][]byte, hash []byte, newMessage *types.NewMessage) {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Add(identifiers, types.BytesToHash(hash), *newMessage)
	}
}

// GetCurrentTime returns the current unix timestamp in milliseconds
func (a *WakuServiceTransport) GetCurrentTime() uint64 {
	return uint64(a.waku.GetCurrentTime().UnixNano() / int64(time.Millisecond))
}

func (a *WakuServiceTransport) Stop() error {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Stop()
	}
	return nil
}

// RequestHistoricMessages requests historic messages for all registered filters.
func (a *WakuServiceTransport) SendMessagesRequest(
	ctx context.Context,
	peerID []byte,
	from, to uint32,
	previousCursor []byte,
) (cursor []byte, err error) {
	topics := make([]types.TopicType, len(a.Filters()))
	for _, f := range a.Filters() {
		topics = append(topics, f.Topic)
	}

	r := createMessagesRequest(from, to, previousCursor, topics)
	r.SetDefaults(a.waku.GetCurrentTime())

	events := make(chan types.EnvelopeEvent, 10)
	sub := a.waku.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	err = a.waku.SendMessagesRequest(peerID, r)
	if err != nil {
		return
	}

	resp, err := a.waitForRequestCompleted(ctx, r.ID, events)
	if err == nil && resp != nil && resp.Error != nil {
		err = resp.Error
	} else if err == nil && resp != nil {
		cursor = resp.Cursor
	}
	return
}

func (a *WakuServiceTransport) waitForRequestCompleted(ctx context.Context, requestID []byte, events chan types.EnvelopeEvent) (*types.MailServerResponse, error) {
	for {
		select {
		case ev := <-events:
			a.logger.Debug(
				"waiting for request completed and received an event",
				zap.Binary("requestID", requestID),
				zap.Any("event", ev),
			)
			if !bytes.Equal(ev.Hash.Bytes(), requestID) {
				continue
			}
			if ev.Event != types.EventMailServerRequestCompleted {
				continue
			}
			data, ok := ev.Data.(*types.MailServerResponse)
			if ok {
				return data, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
