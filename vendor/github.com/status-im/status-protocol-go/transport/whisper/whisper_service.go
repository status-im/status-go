package whisper

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
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

// DEPRECATED
// Use RetrieveAllMessages instead.
func (a *WhisperServiceTransport) RetrieveRawAll() (map[Filter][]*whispertypes.Message, error) {
	return nil, errors.New("not implemented")
}

// DEPRECATED
func (a *WhisperServiceTransport) RetrieveRaw(filterID string) ([]*whispertypes.Message, error) {
	return a.shhAPI.GetFilterMessages(filterID)
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
	newMessage.Topic = whispertypes.TopicType(filter.Topic)

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

	newMessage.Topic = whispertypes.TopicType(filter.Topic)
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
	newMessage.Sig = sigID
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

// MessagesRequest is a RequestMessages() request payload.
type MessagesRequest struct {
	// MailServerPeer is MailServer's enode address.
	MailServerPeer string `json:"mailServerPeer"`

	// From is a lower bound of time range (optional).
	// Default is 24 hours back from now.
	From uint32 `json:"from"`

	// To is a upper bound of time range (optional).
	// Default is now.
	To uint32 `json:"to"`

	// Limit determines the number of messages sent by the mail server
	// for the current paginated request
	Limit uint32 `json:"limit"`

	// Cursor is used as starting point for paginated requests
	Cursor string `json:"cursor"`

	// Topic is a regular Whisper topic.
	// DEPRECATED
	Topic whispertypes.TopicType `json:"topic"`

	// Topics is a list of Whisper topics.
	Topics []whispertypes.TopicType `json:"topics"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string `json:"symKeyID"`

	// Timeout is the time to live of the request specified in seconds.
	// Default is 10 seconds
	Timeout time.Duration `json:"timeout"`

	// Force ensures that requests will bypass enforced delay.
	// TODO(adam): it's currently not handled.
	Force bool `json:"force"`
}

type MessagesResponse struct {
	// Cursor from the response can be used to retrieve more messages
	// for the previous request.
	Cursor string `json:"cursor"`

	// Error indicates that something wrong happened when sending messages
	// to the requester.
	Error error `json:"error"`
}

// RetryConfig specifies configuration for retries with timeout and max amount of retries.
type RetryConfig struct {
	BaseTimeout time.Duration
	// StepTimeout defines duration increase per each retry.
	StepTimeout time.Duration
	MaxRetries  int
}
