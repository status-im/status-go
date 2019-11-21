package whisper

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	statusproto "github.com/status-im/status-go/protocol/types"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type whisperServiceKeysManager struct {
	shh whispertypes.Whisper

	// Identity of the current user.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *whisperServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *whisperServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymKeyMutex.Lock()
	defer m.passToSymKeyMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.shh.AddSymKeyFromPassword(password)
	if err != nil {
		return id, err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *whisperServiceKeysManager) RawSymKey(id string) ([]byte, error) {
	return m.shh.GetSymKey(id)
}

type Option func(*WhisperServiceTransport) error

func SetGenericDiscoveryTopicSupport(val bool) Option {
	return func(t *WhisperServiceTransport) error {
		t.genericDiscoveryTopicEnabled = val
		return nil
	}
}

// WhisperServiceTransport is a transport based on Whisper service.
type WhisperServiceTransport struct {
	shh         whispertypes.Whisper
	shhAPI      whispertypes.PublicWhisperAPI // only PublicWhisperAPI implements logic to send messages
	keysManager *whisperServiceKeysManager
	filters     *filtersManager
	logger      *zap.Logger

	mailservers      []string
	envelopesMonitor *EnvelopesMonitor

	genericDiscoveryTopicEnabled bool
}

// NewWhisperServiceTransport returns a new WhisperServiceTransport.
// TODO: leaving a chat should verify that for a given public key
//       there are no other chats. It may happen that we leave a private chat
//       but still have a public chat for a given public key.
func NewWhisperServiceTransport(
	shh whispertypes.Whisper,
	privateKey *ecdsa.PrivateKey,
	db *sql.DB,
	mailservers []string,
	envelopesMonitorConfig *EnvelopesMonitorConfig,
	logger *zap.Logger,
	opts ...Option,
) (*WhisperServiceTransport, error) {
	filtersManager, err := newFiltersManager(db, shh, privateKey, logger)
	if err != nil {
		return nil, err
	}

	var envelopesMonitor *EnvelopesMonitor
	if envelopesMonitorConfig != nil {
		envelopesMonitor = NewEnvelopesMonitor(shh, *envelopesMonitorConfig)
		envelopesMonitor.Start()
	}

	var shhAPI whispertypes.PublicWhisperAPI
	if shh != nil {
		shhAPI = shh.PublicWhisperAPI()
	}
	t := &WhisperServiceTransport{
		shh:              shh,
		shhAPI:           shhAPI,
		envelopesMonitor: envelopesMonitor,
		keysManager: &whisperServiceKeysManager{
			shh:               shh,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
		filters:     filtersManager,
		mailservers: mailservers,
		logger:      logger.With(zap.Namespace("WhisperServiceTransport")),
	}

	for _, opt := range opts {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (a *WhisperServiceTransport) InitFilters(chatIDs []string, publicKeys []*ecdsa.PublicKey) ([]*Filter, error) {
	return a.filters.Init(chatIDs, publicKeys, a.genericDiscoveryTopicEnabled)
}

func (a *WhisperServiceTransport) Filters() []*Filter {
	return a.filters.Filters()
}

// DEPRECATED
func (a *WhisperServiceTransport) LoadFilters(filters []*Filter) ([]*Filter, error) {
	return a.filters.InitWithFilters(filters, a.genericDiscoveryTopicEnabled)
}

// DEPRECATED
func (a *WhisperServiceTransport) RemoveFilters(filters []*Filter) error {
	return a.filters.Remove(filters...)
}

func (a *WhisperServiceTransport) Reset() error {
	return a.filters.Reset()
}

func (a *WhisperServiceTransport) ProcessNegotiatedSecret(secret whispertypes.NegotiatedSecret) (*Filter, error) {
	filter, err := a.filters.LoadNegotiated(secret)
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (a *WhisperServiceTransport) JoinPublic(chatID string) error {
	_, err := a.filters.LoadPublic(chatID)
	return err
}

func (a *WhisperServiceTransport) LeavePublic(chatID string) error {
	chat := a.filters.Filter(chatID)
	if chat != nil {
		return nil
	}
	return a.filters.Remove(chat)
}

func (a *WhisperServiceTransport) JoinPrivate(publicKey *ecdsa.PublicKey) error {
	_, err := a.filters.LoadDiscovery()
	if err != nil {
		return err
	}
	_, err = a.filters.LoadContactCode(publicKey)
	return err
}

func (a *WhisperServiceTransport) LeavePrivate(publicKey *ecdsa.PublicKey) error {
	filters := a.filters.FiltersByPublicKey(publicKey)
	return a.filters.Remove(filters...)
}

func (a *WhisperServiceTransport) JoinGroup(publicKeys []*ecdsa.PublicKey) error {
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

func (a *WhisperServiceTransport) LeaveGroup(publicKeys []*ecdsa.PublicKey) error {
	for _, publicKey := range publicKeys {
		filters := a.filters.FiltersByPublicKey(publicKey)
		if err := a.filters.Remove(filters...); err != nil {
			return err
		}
	}
	return nil
}

type Message struct {
	Message *whispertypes.Message
	Public  bool
}

func (a *WhisperServiceTransport) RetrieveAllMessages() ([]Message, error) {
	var messages []Message

	for _, filter := range a.filters.Filters() {
		filterMsgs, err := a.shhAPI.GetFilterMessages(filter.FilterID)
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

func (a *WhisperServiceTransport) RetrievePublicMessages(chatID string) ([]*whispertypes.Message, error) {
	filter, err := a.filters.LoadPublic(chatID)
	if err != nil {
		return nil, err
	}

	return a.shhAPI.GetFilterMessages(filter.FilterID)
}

func (a *WhisperServiceTransport) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*whispertypes.Message, error) {
	chats := a.filters.FiltersByPublicKey(publicKey)
	discoveryChats, err := a.filters.Init(nil, nil, true)
	if err != nil {
		return nil, err
	}

	var result []*whispertypes.Message

	for _, chat := range append(chats, discoveryChats...) {
		filterMsgs, err := a.shhAPI.GetFilterMessages(chat.FilterID)
		if err != nil {
			return nil, err
		}

		result = append(result, filterMsgs...)
	}

	return result, nil
}

func (a *WhisperServiceTransport) RetrieveRawAll() (map[Filter][]*whispertypes.Message, error) {
	result := make(map[Filter][]*whispertypes.Message)

	allFilters := a.filters.Filters()
	for _, filter := range allFilters {
		f := a.shh.GetFilter(filter.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		msgs, err := a.shhAPI.GetFilterMessages(filter.FilterID)
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
func (a *WhisperServiceTransport) SendPublic(ctx context.Context, newMessage *whispertypes.NewMessage, chatName string) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadPublic(chatName)
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = filter.SymKeyID
	newMessage.Topic = filter.Topic

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) SendPrivateWithSharedSecret(ctx context.Context, newMessage *whispertypes.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadNegotiated(whispertypes.NegotiatedSecret{
		PublicKey: publicKey,
		Key:       secret,
	})
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = filter.SymKeyID
	newMessage.Topic = whispertypes.TopicType(filter.Topic)
	newMessage.PublicKey = nil

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) SendPrivateWithPartitioned(ctx context.Context, newMessage *whispertypes.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadPartitioned(publicKey)
	if err != nil {
		return nil, err
	}

	newMessage.Topic = filter.Topic
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) SendPrivateOnDiscovery(ctx context.Context, newMessage *whispertypes.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	// There is no need to load any chat
	// because listening on the discovery topic
	// is done automatically.
	// TODO: change this anyway, it should be explicit
	// and idempotent.

	newMessage.Topic = whispertypes.BytesToTopic(
		ToTopic(discoveryTopic),
	)
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) addSig(newMessage *whispertypes.NewMessage) error {
	sigID, err := a.keysManager.AddOrGetKeyPair(a.keysManager.privateKey)
	if err != nil {
		return err
	}
	newMessage.SigID = sigID
	return nil
}

func (a *WhisperServiceTransport) Track(identifiers [][]byte, hash []byte, newMessage *whispertypes.NewMessage) {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Add(identifiers, statusproto.BytesToHash(hash), *newMessage)
	}
}

func (a *WhisperServiceTransport) Stop() error {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Stop()
	}
	return nil
}

// RequestHistoricMessages requests historic messages for all registered filters.
func (a *WhisperServiceTransport) SendMessagesRequest(
	ctx context.Context,
	peerID []byte,
	from, to uint32,
	previousCursor []byte,
) (cursor []byte, err error) {
	topics := make([]whispertypes.TopicType, len(a.Filters()))
	for _, f := range a.Filters() {
		topics = append(topics, f.Topic)
	}

	r := createMessagesRequest(from, to, previousCursor, topics)
	r.SetDefaults(a.shh.GetCurrentTime())

	events := make(chan whispertypes.EnvelopeEvent, 10)
	sub := a.shh.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	err = a.shh.SendMessagesRequest(peerID, r)
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

func (a *WhisperServiceTransport) waitForRequestCompleted(ctx context.Context, requestID []byte, events chan whispertypes.EnvelopeEvent) (*whispertypes.MailServerResponse, error) {
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
			if ev.Event != whispertypes.EventMailServerRequestCompleted {
				continue
			}
			data, ok := ev.Data.(*whispertypes.MailServerResponse)
			if ok {
				return data, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
