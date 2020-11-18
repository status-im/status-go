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

type Option func(*Transport) error

// Transport is a transport based on Whisper service.
type Transport struct {
	waku        types.Waku
	api         types.PublicWakuAPI // only PublicWakuAPI implements logic to send messages
	keysManager *wakuServiceKeysManager
	filters     *transport.FiltersManager
	logger      *zap.Logger

	mailservers      []string
	envelopesMonitor *EnvelopesMonitor
}

// NewTransport returns a new Transport.
// TODO: leaving a chat should verify that for a given public key
//       there are no other chats. It may happen that we leave a private chat
//       but still have a public chat for a given public key.
func NewTransport(
	waku types.Waku,
	privateKey *ecdsa.PrivateKey,
	db *sql.DB,
	mailservers []string,
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig,
	logger *zap.Logger,
	opts ...Option,
) (*Transport, error) {
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
	t := &Transport{
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
		logger:      logger.With(zap.Namespace("Transport")),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (a *Transport) InitFilters(chatIDs []string, publicKeys []*ecdsa.PublicKey) ([]*transport.Filter, error) {
	return a.filters.Init(chatIDs, publicKeys)
}

func (a *Transport) InitPublicFilters(chatIDs []string) ([]*transport.Filter, error) {
	return a.filters.InitPublicFilters(chatIDs)
}

func (a *Transport) Filters() []*transport.Filter {
	return a.filters.Filters()
}

func (a *Transport) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return a.filters.InitWithFilters(filters)
}

func (a *Transport) RemoveFilters(filters []*transport.Filter) error {
	return a.filters.Remove(filters...)
}

func (a *Transport) RemoveFilterByChatID(chatID string) error {
	return a.filters.RemoveFilterByChatID(chatID)
}

func (a *Transport) ResetFilters() error {
	return a.filters.Reset()
}

func (a *Transport) ProcessNegotiatedSecret(secret types.NegotiatedSecret) (*transport.Filter, error) {
	filter, err := a.filters.LoadNegotiated(secret)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (a *Transport) JoinPublic(chatID string) error {
	_, err := a.filters.LoadPublic(chatID)
	return err
}

func (a *Transport) LeavePublic(chatID string) error {
	chat := a.filters.Filter(chatID)
	if chat != nil {
		return nil
	}
	return a.filters.Remove(chat)
}

func (a *Transport) JoinPrivate(publicKey *ecdsa.PublicKey) error {
	_, err := a.filters.LoadDiscovery()
	if err != nil {
		return err
	}
	_, err = a.filters.LoadContactCode(publicKey)
	return err
}

func (a *Transport) LeavePrivate(publicKey *ecdsa.PublicKey) error {
	filters := a.filters.FiltersByPublicKey(publicKey)
	return a.filters.Remove(filters...)
}

func (a *Transport) JoinGroup(publicKeys []*ecdsa.PublicKey) error {
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

func (a *Transport) LeaveGroup(publicKeys []*ecdsa.PublicKey) error {
	for _, publicKey := range publicKeys {
		filters := a.filters.FiltersByPublicKey(publicKey)
		if err := a.filters.Remove(filters...); err != nil {
			return err
		}
	}
	return nil
}

func (a *Transport) RetrieveRawAll() (map[transport.Filter][]*types.Message, error) {
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
func (a *Transport) SendPublic(ctx context.Context, newMessage *types.NewMessage, chatName string) ([]byte, error) {
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

func (a *Transport) SendPrivateWithSharedSecret(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
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

func (a *Transport) SendPrivateWithPartitioned(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadPartitioned(publicKey, a.keysManager.privateKey, false)
	if err != nil {
		return nil, err
	}

	newMessage.Topic = filter.Topic
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.api.Post(ctx, *newMessage)
}

func (a *Transport) LoadKeyFilters(key *ecdsa.PrivateKey) (*transport.Filter, error) {
	return a.filters.LoadPartitioned(&key.PublicKey, key, true)
}

func (a *Transport) SendPrivateOnDiscovery(ctx context.Context, newMessage *types.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
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

func (a *Transport) addSig(newMessage *types.NewMessage) error {
	sigID, err := a.keysManager.AddOrGetKeyPair(a.keysManager.privateKey)
	if err != nil {
		return err
	}
	newMessage.SigID = sigID
	return nil
}

func (a *Transport) Track(identifiers [][]byte, hash []byte, newMessage *types.NewMessage) {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Add(identifiers, types.BytesToHash(hash), *newMessage)
	}
}

// GetCurrentTime returns the current unix timestamp in milliseconds
func (a *Transport) GetCurrentTime() uint64 {
	return uint64(a.waku.GetCurrentTime().UnixNano() / int64(time.Millisecond))
}

func (a *Transport) MaxMessageSize() uint32 {
	return a.waku.MaxMessageSize()
}

func (a *Transport) Stop() error {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Stop()
	}
	return nil
}

// RequestHistoricMessages requests historic messages for all registered filters.
func (a *Transport) SendMessagesRequest(
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

func (a *Transport) waitForRequestCompleted(ctx context.Context, requestID []byte, events chan types.EnvelopeEvent) (*types.MailServerResponse, error) {
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

func (a *Transport) SetEnvelopeEventsHandler(handler transport.EnvelopeEventsHandler) error {
	if a.envelopesMonitor == nil {
		return errors.New("Current transport has no envelopes monitor")
	}
	a.envelopesMonitor.handler = handler
	return nil
}
