package whisper

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/mailserver"
	whisper "github.com/status-im/whisper/whisperv6"
	"go.uber.org/zap"

	"github.com/status-im/status-protocol-go/transport/whisper/filter"
)

const (
	// defaultRequestTimeout is the default request timeout in seconds
	defaultRequestTimeout = 10
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

// WhisperServiceTransport is a transport based on Whisper service.
type WhisperServiceTransport struct {
	node        Server
	shh         *whisper.Whisper
	shhAPI      *whisper.PublicWhisperAPI // only PublicWhisperAPI implements logic to send messages
	keysManager *whisperServiceKeysManager
	chats       *filter.ChatsManager
	logger      *zap.Logger

	mailservers             []string
	selectedMailServerEnode string
}

// NewWhisperService returns a new WhisperServiceTransport.
func NewWhisperServiceTransport(
	node Server,
	shh *whisper.Whisper,
	privateKey *ecdsa.PrivateKey,
	db *sql.DB,
	mailservers []string,
	logger *zap.Logger,
) (*WhisperServiceTransport, error) {
	chats, err := filter.New(db, shh, privateKey, logger)
	if err != nil {
		return nil, err
	}

	return &WhisperServiceTransport{
		node:   node,
		shh:    shh,
		shhAPI: whisper.NewPublicWhisperAPI(shh),
		keysManager: &whisperServiceKeysManager{
			shh:               shh,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
		chats:       chats,
		mailservers: mailservers,
		logger:      logger.With(zap.Namespace("WhisperServiceTransport")),
	}, nil
}

// DEPRECATED
func (a *WhisperServiceTransport) LoadFilters(chats []*filter.Chat, genericDiscoveryTopicEnabled bool) ([]*filter.Chat, error) {
	return a.chats.InitWithChats(chats, genericDiscoveryTopicEnabled)
}

// DEPRECATED
func (a *WhisperServiceTransport) RemoveFilters(chats []*filter.Chat) error {
	return a.chats.Remove(chats...)
}

func (a *WhisperServiceTransport) Reset() error {
	return a.chats.Reset()
}

func (a *WhisperServiceTransport) ProcessNegotiatedSecret(secret filter.NegotiatedSecret) error {
	_, err := a.chats.LoadNegotiated(secret)
	return err
}

func (a *WhisperServiceTransport) JoinPublic(chatID string) error {
	_, err := a.chats.LoadPublic(chatID)
	return err
}

func (a *WhisperServiceTransport) LeavePublic(chatID string) error {
	chat := a.chats.ChatByID(chatID)
	if chat != nil {
		return nil
	}
	return a.chats.Remove(chat)
}

func (a *WhisperServiceTransport) JoinPrivate(publicKey *ecdsa.PublicKey) error {
	_, err := a.chats.LoadContactCode(publicKey)
	return err
}

func (a *WhisperServiceTransport) LeavePrivate(publicKey *ecdsa.PublicKey) error {
	chats := a.chats.ChatsByPublicKey(publicKey)
	return a.chats.Remove(chats...)
}

type ChatMessages struct {
	Messages []*whisper.ReceivedMessage
	Public   bool
	ChatID   string
}

func (a *WhisperServiceTransport) RetrieveAllMessages() ([]ChatMessages, error) {
	chatMessages := make(map[string]ChatMessages)

	for _, chat := range a.chats.Chats() {
		f := a.shh.GetFilter(chat.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		messages := chatMessages[chat.ChatID]
		messages.ChatID = chat.ChatID
		messages.Public = chat.IsPublic()
		messages.Messages = append(messages.Messages, f.Retrieve()...)
	}

	var result []ChatMessages
	for _, messages := range chatMessages {
		result = append(result, messages)
	}
	return result, nil
}

func (a *WhisperServiceTransport) RetrievePublicMessages(chatID string) ([]*whisper.ReceivedMessage, error) {
	chat, err := a.chats.LoadPublic(chatID)
	if err != nil {
		return nil, err
	}

	f := a.shh.GetFilter(chat.FilterID)
	if f == nil {
		return nil, errors.New("failed to return a filter")
	}

	return f.Retrieve(), nil
}

func (a *WhisperServiceTransport) RetrievePrivateMessages(publicKey *ecdsa.PublicKey) ([]*whisper.ReceivedMessage, error) {
	chats := a.chats.ChatsByPublicKey(publicKey)
	discoveryChats, err := a.chats.Init(nil, nil, true)
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
func (a *WhisperServiceTransport) RetrieveRawAll() (map[filter.Chat][]*whisper.ReceivedMessage, error) {
	result := make(map[filter.Chat][]*whisper.ReceivedMessage)

	allChats := a.chats.Chats()
	for _, chat := range allChats {
		f := a.shh.GetFilter(chat.FilterID)
		if f == nil {
			return nil, errors.New("failed to return a filter")
		}

		result[*chat] = append(result[*chat], f.Retrieve()...)
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
// For public chats, chat name is used as an ID as well as
// a topic.
func (a *WhisperServiceTransport) SendPublic(ctx context.Context, newMessage whisper.NewMessage, chatName string) ([]byte, error) {
	if err := a.addSig(&newMessage); err != nil {
		return nil, err
	}

	chat, err := a.chats.LoadPublic(chatName)
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = chat.SymKeyID
	newMessage.Topic = chat.Topic

	return a.shhAPI.Post(ctx, newMessage)
}

func (a *WhisperServiceTransport) SendPrivateWithSharedSecret(ctx context.Context, newMessage whisper.NewMessage, publicKey *ecdsa.PublicKey, secret []byte) ([]byte, error) {
	if err := a.addSig(&newMessage); err != nil {
		return nil, err
	}

	chat, err := a.chats.LoadNegotiated(filter.NegotiatedSecret{
		PublicKey: publicKey,
		Key:       secret,
	})
	if err != nil {
		return nil, err
	}

	newMessage.SymKeyID = chat.SymKeyID
	newMessage.Topic = chat.Topic
	newMessage.PublicKey = nil

	return a.shhAPI.Post(ctx, newMessage)
}

func (a *WhisperServiceTransport) SendPrivateWithPartitioned(ctx context.Context, newMessage whisper.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(&newMessage); err != nil {
		return nil, err
	}

	chat, err := a.chats.LoadPartitioned(publicKey)
	if err != nil {
		return nil, err
	}

	newMessage.Topic = chat.Topic
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.shhAPI.Post(ctx, newMessage)
}

func (a *WhisperServiceTransport) SendPrivateOnDiscovery(ctx context.Context, newMessage whisper.NewMessage, publicKey *ecdsa.PublicKey) ([]byte, error) {
	if err := a.addSig(&newMessage); err != nil {
		return nil, err
	}

	// There is no need to load any chat
	// because listening on the discovery topic
	// is done automatically.
	// TODO: change this anyway, it should be explicit
	// and idempotent.

	newMessage.Topic = whisper.BytesToTopic(
		filter.ToTopic(filter.DiscoveryTopic),
	)
	newMessage.PublicKey = crypto.FromECDSAPub(publicKey)

	return a.shhAPI.Post(ctx, newMessage)
}

func (a *WhisperServiceTransport) addSig(newMessage *whisper.NewMessage) error {
	sigID, err := a.keysManager.AddOrGetKeyPair(a.keysManager.privateKey)
	if err != nil {
		return err
	}
	newMessage.Sig = sigID
	return nil
}

// Request requests messages from mail servers.
func (a *WhisperServiceTransport) Request(ctx context.Context, options RequestOptions) error {
	// TODO: remove from here. MailServerEnode must be provided in the params.
	enode, err := a.selectAndAddMailServer()
	if err != nil {
		return err
	}

	keyID, err := a.keysManager.AddOrGetSymKeyFromPassword(options.Password)
	if err != nil {
		return err
	}

	req, err := createRequestMessagesParam(enode, keyID, options)
	if err != nil {
		return err
	}

	_, err = a.requestMessages(ctx, req, true)
	return err
}

func (a *WhisperServiceTransport) requestMessages(ctx context.Context, req MessagesRequest, followCursor bool) (resp MessagesResponse, err error) {
	logger := a.logger.With(zap.String("site", "requestMessages"))

	logger.Debug("request for a chunk", zap.Uint32("message-limit", req.Limit))

	start := time.Now()
	resp, err = a.requestMessagesWithRetry(RetryConfig{
		BaseTimeout: time.Second * 10,
		StepTimeout: time.Second,
		MaxRetries:  3,
	}, req)
	if err != nil {
		logger.Error("failed requesting messages", zap.Error(err))
		return
	}

	logger.Debug("message delivery summary",
		zap.Uint32("message-limit", req.Limit),
		zap.Duration("duration", time.Since(start)),
		zap.Any("response", resp),
	)

	if resp.Error != nil {
		err = resp.Error
		return
	}
	if !followCursor || resp.Cursor == "" {
		return
	}

	req.Cursor = resp.Cursor
	logger.Debug("requesting messages with cursor", zap.String("cursor", req.Cursor))
	return a.requestMessages(ctx, req, true)
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

func (r *MessagesRequest) setDefaults(now time.Time) {
	// set From and To defaults
	if r.To == 0 {
		r.To = uint32(now.UTC().Unix())
	}

	if r.From == 0 {
		oneDay := uint32(86400) // -24 hours
		if r.To < oneDay {
			r.From = 0
		} else {
			r.From = r.To - oneDay
		}
	}

	if r.Timeout == 0 {
		r.Timeout = defaultRequestTimeout
	}
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

func (a *WhisperServiceTransport) requestMessagesWithRetry(conf RetryConfig, r MessagesRequest) (MessagesResponse, error) {
	var (
		resp      MessagesResponse
		requestID hexutil.Bytes
		err       error
		retries   int
	)

	logger := a.logger.With(zap.String("site", "requestMessagesWithRetry"))

	events := make(chan whisper.EnvelopeEvent, 10)

	for retries <= conf.MaxRetries {
		sub := a.shh.SubscribeEnvelopeEvents(events)
		r.Timeout = conf.BaseTimeout + conf.StepTimeout*time.Duration(retries)
		timeout := r.Timeout
		// FIXME this weird conversion is required because MessagesRequest expects seconds but defines time.Duration
		r.Timeout = time.Duration(int(r.Timeout.Seconds()))
		requestID, err = a.requestMessagesSync(context.Background(), r)
		if err != nil {
			sub.Unsubscribe()
			return resp, err
		}

		mailServerResp, err := waitForExpiredOrCompleted(common.BytesToHash(requestID), events, timeout)
		sub.Unsubscribe()
		if err == nil {
			resp.Cursor = hex.EncodeToString(mailServerResp.Cursor)
			resp.Error = mailServerResp.Error
			return resp, nil
		}
		retries++
		logger.Warn("requestMessagesSync failed, retrying", zap.Int("retries", retries), zap.Error(err))
	}
	return resp, fmt.Errorf("failed to request messages after %d retries", retries)
}

// RequestMessages sends a request for historic messages to a MailServer.
func (a *WhisperServiceTransport) requestMessagesSync(_ context.Context, r MessagesRequest) (hexutil.Bytes, error) {
	now := a.shh.GetCurrentTime()
	r.setDefaults(now)

	if r.From > r.To {
		return nil, fmt.Errorf("Query range is invalid: from > to (%d > %d)", r.From, r.To)
	}

	// TODO: bring mailserverspackage here
	mailServerNode, err := enode.ParseV4(r.MailServerPeer)
	if err != nil {
		return nil, fmt.Errorf("invalid MailServerPeer: %v", err)
	}

	var (
		symKey    []byte
		publicKey *ecdsa.PublicKey
	)

	if r.SymKeyID != "" {
		symKey, err = a.shh.GetSymKey(r.SymKeyID)
		if err != nil {
			return nil, fmt.Errorf("invalid SymKeyID: %v", err)
		}
	} else {
		publicKey = mailServerNode.Pubkey()
	}

	payload, err := makeMessagesRequestPayload(r)
	if err != nil {
		return nil, err
	}

	envelope, err := makeEnvelop(
		payload,
		symKey,
		publicKey,
		a.node.NodeID(),
		a.shh.MinPow(),
		now,
	)
	if err != nil {
		return nil, err
	}
	hash := envelope.Hash()

	if err := a.shh.RequestHistoricMessagesWithTimeout(mailServerNode.ID().Bytes(), envelope, r.Timeout*time.Second); err != nil {
		return nil, err
	}

	return hash[:], nil
}

func (a *WhisperServiceTransport) selectAndAddMailServer() (string, error) {
	logger := a.logger.With(zap.String("site", "selectAndAddMailServer"))

	var enodeAddr string
	if a.selectedMailServerEnode != "" {
		enodeAddr = a.selectedMailServerEnode
	} else {
		if len(a.mailservers) == 0 {
			return "", ErrNoMailservers
		}
		enodeAddr = randomItem(a.mailservers)
	}
	logger.Debug("dialing mail server", zap.String("enode", enodeAddr))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := dial(ctx, a.node, enodeAddr, dialOpts{PollInterval: 200 * time.Millisecond})
	cancel()
	if err == nil {
		a.selectedMailServerEnode = enodeAddr
		return enodeAddr, nil
	}
	return "", fmt.Errorf("peer %s failed to connect: %v", enodeAddr, err)
}

func createRequestMessagesParam(enode, symKeyID string, options RequestOptions) (MessagesRequest, error) {
	req := MessagesRequest{
		MailServerPeer: enode,
		From:           uint32(options.From),  // TODO: change to int in status-go
		To:             uint32(options.To),    // TODO: change to int in status-go
		Limit:          uint32(options.Limit), // TODO: change to int in status-go
		SymKeyID:       symKeyID,
		Topics:         options.Topics,
	}

	return req, nil
}

func waitForExpiredOrCompleted(requestID common.Hash, events chan whisper.EnvelopeEvent, timeout time.Duration) (*whisper.MailServerResponse, error) {
	expired := fmt.Errorf("request %x expired", requestID)
	after := time.NewTimer(timeout)
	defer after.Stop()
	for {
		var ev whisper.EnvelopeEvent
		select {
		case ev = <-events:
		case <-after.C:
			return nil, expired
		}
		if ev.Hash != requestID {
			continue
		}
		switch ev.Event {
		case whisper.EventMailServerRequestCompleted:
			data, ok := ev.Data.(*whisper.MailServerResponse)
			if ok {
				return data, nil
			}
			return nil, errors.New("invalid event data type")
		case whisper.EventMailServerRequestExpired:
			return nil, expired
		}
	}
}

// makeMessagesRequestPayload makes a specific payload for MailServer
// to request historic messages.
func makeMessagesRequestPayload(r MessagesRequest) ([]byte, error) {
	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %v", err)
	}

	if len(cursor) > 0 && len(cursor) != mailserver.CursorLength {
		return nil, fmt.Errorf("invalid cursor size: expected %d but got %d", mailserver.CursorLength, len(cursor))
	}

	payload := mailserver.MessagesRequestPayload{
		Lower:  r.From,
		Upper:  r.To,
		Bloom:  createBloomFilter(r),
		Limit:  r.Limit,
		Cursor: cursor,
		// Client must tell the MailServer if it supports batch responses.
		// This can be removed in the future.
		Batch: true,
	}

	return rlp.EncodeToBytes(payload)
}

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelop(
	payload []byte,
	symKey []byte,
	publicKey *ecdsa.PublicKey,
	nodeID *ecdsa.PrivateKey,
	pow float64,
	now time.Time,
) (*whisper.Envelope, error) {
	params := whisper.MessageParams{
		PoW:      pow,
		Payload:  payload,
		WorkTime: DefaultWhisperMessage().PowTime,
		Src:      nodeID,
	}
	// Either symKey or public key is required.
	// This condition is verified in `message.Wrap()` method.
	if len(symKey) > 0 {
		params.KeySym = symKey
	} else if publicKey != nil {
		params.Dst = publicKey
	}
	message, err := whisper.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	return message.Wrap(&params, now)
}

func createBloomFilter(r MessagesRequest) []byte {
	if len(r.Topics) > 0 {
		return topicsToBloom(r.Topics...)
	}

	return whisper.TopicToBloom(r.Topic)
}

func topicsToBloom(topics ...whisper.TopicType) []byte {
	i := new(big.Int)
	for _, topic := range topics {
		bloom := whisper.TopicToBloom(topic)
		i.Or(i, new(big.Int).SetBytes(bloom[:]))
	}

	combined := make([]byte, whisper.BloomFilterSize)
	data := i.Bytes()
	copy(combined[whisper.BloomFilterSize-len(data):], data[:])

	return combined
}
