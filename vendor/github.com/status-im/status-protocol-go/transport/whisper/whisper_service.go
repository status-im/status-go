package whisper

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"
	"go.uber.org/zap"
)

var (
	// ErrNoMailservers returned if there is no configured mailservers that can be used.
	ErrNoMailservers = errors.New("no configured mailservers")
)

type whisperServiceKeysManager struct {
	shh *whisper.Whisper

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
	shh         *whisper.Whisper
	shhAPI      *whisper.PublicWhisperAPI // only PublicWhisperAPI implements logic to send messages
	keysManager *whisperServiceKeysManager
	filters     *filtersManager
	logger      *zap.Logger

	mailservers      []string
	envelopesMonitor *EnvelopesMonitor

	genericDiscoveryTopicEnabled bool
}

// NewWhisperService returns a new WhisperServiceTransport.
func NewWhisperServiceTransport(
	shh *whisper.Whisper,
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
		envelopesMonitor = NewEnvelopesMonitor(shh, envelopesMonitorConfig)
		envelopesMonitor.Start()
	}

	t := &WhisperServiceTransport{
		shh:              shh,
		shhAPI:           whisper.NewPublicWhisperAPI(shh),
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

func (a *WhisperServiceTransport) ProcessNegotiatedSecret(secret NegotiatedSecret) (*Filter, error) {
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

type ChatMessages struct {
	Messages []*whisper.ReceivedMessage
	Public   bool
	ChatID   string
}

func (a *WhisperServiceTransport) RetrieveAllMessages() ([]ChatMessages, error) {
	chatMessages := make(map[string]ChatMessages)

	for _, filter := range a.filters.Filters() {
		f := a.shh.GetFilter(filter.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		ch := chatMessages[filter.ChatID]
		ch.ChatID = filter.ChatID
		ch.Public = filter.IsPublic()
		ch.Messages = append(ch.Messages, f.Retrieve()...)
		chatMessages[filter.ChatID] = ch
	}

	var result []ChatMessages
	for _, messages := range chatMessages {
		result = append(result, messages)
	}
	return result, nil
}

func (a *WhisperServiceTransport) RetrievePublicMessages(chatID string) ([]*whisper.ReceivedMessage, error) {
	filter, err := a.filters.LoadPublic(chatID)
	if err != nil {
		return nil, err
	}

	f := a.shh.GetFilter(filter.FilterID)
	if f == nil {
		return nil, errors.New("failed to return a filter")
	}

	return f.Retrieve(), nil
}

func (a *WhisperServiceTransport) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*whisper.ReceivedMessage, error) {
	chats := a.filters.FiltersByPublicKey(publicKey)
	discoveryChats, err := a.filters.Init(nil, nil, true)
	if err != nil {
		return nil, err
	}

	var result []*whisper.ReceivedMessage

	for _, chat := range append(chats, discoveryChats...) {
		f := a.shh.GetFilter(chat.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		result = append(result, f.Retrieve()...)
	}

	return result, nil
}

// DEPRECATED
func (a *WhisperServiceTransport) RetrieveRawAll() (map[Filter][]*whisper.ReceivedMessage, error) {
	result := make(map[Filter][]*whisper.ReceivedMessage)

	allFilters := a.filters.Filters()
	for _, filter := range allFilters {
		f := a.shh.GetFilter(filter.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		result[*filter] = append(result[*filter], f.Retrieve()...)
	}

	return result, nil
}

// DEPRECATED
func (a *WhisperServiceTransport) RetrieveRaw(filterID string) ([]*whisper.ReceivedMessage, error) {
	f := a.shh.GetFilter(filterID)
	if f == nil {
		return nil, errors.New("failed to return a filter")
	}
	return f.Retrieve(), nil
}

// SendPublic sends a new message using the Whisper service.
// For public filters, chat name is used as an ID as well as
// a topic.
func (a *WhisperServiceTransport) SendPublic(ctx context.Context, newMessage *whisper.NewMessage, chatName string) ([]byte, error) {
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

func (a *WhisperServiceTransport) SendPrivateWithSharedSecret(ctx context.Context, newMessage *whisper.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	filter, err := a.filters.LoadNegotiated(NegotiatedSecret{
		PublicKey: publicKey,
		Key:       secret,
	})
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = filter.SymKeyID
	newMessage.Topic = filter.Topic
	newMessage.PublicKey = nil

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) SendPrivateWithPartitioned(ctx context.Context, newMessage *whisper.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
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

func (a *WhisperServiceTransport) SendPrivateOnDiscovery(ctx context.Context, newMessage *whisper.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(newMessage); err != nil {
		return nil, err
	}

	// There is no need to load any chat
	// because listening on the discovery topic
	// is done automatically.
	// TODO: change this anyway, it should be explicit
	// and idempotent.

	newMessage.Topic = whisper.BytesToTopic(
		ToTopic(discoveryTopic),
	)
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.shhAPI.Post(ctx, *newMessage)
}

func (a *WhisperServiceTransport) addSig(newMessage *whisper.NewMessage) error {
	sigID, err := a.keysManager.AddOrGetKeyPair(a.keysManager.privateKey)
	if err != nil {
		return err
	}
	newMessage.Sig = sigID
	return nil
}

func (a *WhisperServiceTransport) Track(identifiers [][]byte, hash []byte, newMessage whisper.NewMessage) {
	if a.envelopesMonitor != nil {
		a.envelopesMonitor.Add(identifiers, common.BytesToHash(hash), newMessage)
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
	Topic whisper.TopicType `json:"topic"`

	// Topics is a list of Whisper topics.
	Topics []whisper.TopicType `json:"topics"`

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
