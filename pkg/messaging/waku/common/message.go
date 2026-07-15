package common

import (
	"crypto/ecdsa"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/logutils"
)

// MessageType represents where this message comes from
type MessageType = string

const (
	RelayedMessageType MessageType = "relay"
	StoreMessageType   MessageType = "store"
	SendMessageType    MessageType = "send"
	MissingMessageType MessageType = "missing"
)

// MessageParams specifies the exact way a message should be wrapped
// into an Envelope.
type MessageParams struct {
	Src     *ecdsa.PrivateKey
	Dst     *ecdsa.PublicKey
	KeySym  []byte
	Topic   TopicType
	Payload []byte
	Padding []byte
}

// ReceivedMessage represents a data packet to be received through the
// WakuV2 protocol and successfully decrypted.
type ReceivedMessage struct {
	Envelope Envelope // Wrapped Waku Message

	MsgType MessageType

	Data      []byte
	Padding   []byte
	Signature []byte

	Sent uint32           // Time when the message was posted into the network in seconds
	Src  *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Dst  *ecdsa.PublicKey // Message recipient (identity used to decode the message)

	PubsubTopic  string
	ContentTopic TopicType

	SymKeyHash common.Hash // The Keccak256Hash of the key

	hash common.Hash
}

// EnvelopeError code and optional description of the error.
type EnvelopeError struct {
	Hash        common.Hash
	Code        uint
	Description string
}

// MessagesResponse sent as a response after processing batch of envelopes.
type MessagesResponse struct {
	// Hash is a hash of all envelopes sent in the single batch.
	Hash common.Hash
	// Per envelope error.
	Errors []EnvelopeError
}

func (msg *ReceivedMessage) isSymmetricEncryption() bool {
	return msg.SymKeyHash != common.Hash{}
}

func (msg *ReceivedMessage) isAsymmetricEncryption() bool {
	return msg.Dst != nil
}

// MessageStore defines interface for temporary message store.
type MessageStore interface {
	Add(*ReceivedMessage) error
	Pop() ([]*ReceivedMessage, error)
	// Count reports how many messages are currently held (added but not yet popped).
	// It is the per-filter contribution to the hash-first body-fetch backpressure
	// signal (issue #21470-hf).
	Count() int
}

// NewMemoryMessageStore returns pointer to an instance of the MemoryMessageStore.
func NewMemoryMessageStore() *MemoryMessageStore {
	return &MemoryMessageStore{
		messages: map[common.Hash]*ReceivedMessage{},
	}
}

// MemoryMessageStore represents messages stored in a memory hash table.
type MemoryMessageStore struct {
	mu       sync.Mutex
	messages map[common.Hash]*ReceivedMessage
}

func NewReceivedMessage(env Envelope, msgType MessageType) *ReceivedMessage {
	ct, err := ExtractTopicFromContentTopic(env.Message().ContentTopic)
	if err != nil {
		logutils.ZapLogger().Debug("failed to extract content topic from message",
			zap.String("topic", env.Message().ContentTopic),
			zap.Error(err),
		)
		return nil
	}

	return &ReceivedMessage{
		Envelope:     env,
		MsgType:      msgType,
		Sent:         uint32(env.Message().GetTimestamp() / int64(time.Second)),
		ContentTopic: ct,
		PubsubTopic:  env.PubsubTopic(),
	}
}

// Hash returns the SHA3 hash of the envelope, calculating it if not yet done.
func (msg *ReceivedMessage) Hash() common.Hash {
	if (msg.hash == common.Hash{}) {
		msg.hash = common.BytesToHash(msg.Envelope.Hash().Bytes())
	}
	return msg.hash
}

// Add adds message to store.
func (store *MemoryMessageStore) Add(msg *ReceivedMessage) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exist := store.messages[msg.Hash()]; !exist {
		store.messages[msg.Hash()] = msg
	}
	return nil
}

// Count returns how many messages are currently held in the store.
func (store *MemoryMessageStore) Count() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.messages)
}

// Pop returns all available messages and cleans the store.
func (store *MemoryMessageStore) Pop() ([]*ReceivedMessage, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	all := make([]*ReceivedMessage, 0, len(store.messages))
	for hash, msg := range store.messages {
		delete(store.messages, hash)
		all = append(all, msg)
	}
	return all, nil
}

// Open tries to decrypt an message, and populates the message fields in case of success.
func (msg *ReceivedMessage) Open(watcher *Filter) (result *ReceivedMessage) {
	if watcher == nil {
		return nil
	}

	// The API interface forbids filters doing both symmetric and asymmetric encryption.
	if watcher.expectsAsymmetricEncryption() && watcher.expectsSymmetricEncryption() {
		return nil
	}

	// TODO: should we update msg instead of creating a new received message?
	result = new(ReceivedMessage)

	keyInfo := new(payload.KeyInfo)
	if watcher.expectsAsymmetricEncryption() {
		keyInfo.Kind = payload.Asymmetric
		keyInfo.PrivKey = watcher.KeyAsym
		msg.Dst = &watcher.KeyAsym.PublicKey
	} else if watcher.expectsSymmetricEncryption() {
		keyInfo.Kind = payload.Symmetric
		keyInfo.SymKey = watcher.KeySym
		msg.SymKeyHash = crypto.Keccak256Hash(watcher.KeySym)
	}

	wakuMsg := msg.Envelope.Message()

	// Migration to retire the WakuMessage `version` field (status-go#7499):
	// always decode incoming payloads as WakuV1-encrypted, treating version=0
	// the same as version=1. status-go only ever emits version=1, so a
	// version=0 envelope is never genuine plaintext; once later releases start
	// publishing version=0 with v1-encoded payloads, they must still decrypt
	// here. We clone before overriding so the original envelope (hash, storage)
	// is untouched, and only force this when the filter has a decryption key —
	// keyless filters keep the legacy "return payload as-is" behaviour.
	if wakuMsg.GetVersion() == 0 && (keyInfo.Kind == payload.Symmetric || keyInfo.Kind == payload.Asymmetric) {
		wakuMsg = proto.Clone(wakuMsg).(*pb.WakuMessage)
		forcedVersion := uint32(1)
		wakuMsg.Version = &forcedVersion
	}

	raw, err := payload.DecodePayload(wakuMsg, keyInfo)

	if err != nil {
		logutils.ZapLogger().Error("failed to decode message", zap.Error(err))
		return nil
	}

	result.Envelope = msg.Envelope
	result.Data = raw.Data
	result.Padding = raw.Padding
	result.Signature = raw.Signature
	result.Src = raw.PubKey
	result.SymKeyHash = msg.SymKeyHash
	result.Dst = msg.Dst
	result.Sent = uint32(msg.Envelope.Message().GetTimestamp() / int64(time.Second))

	ct, err := ExtractTopicFromContentTopic(msg.Envelope.Message().ContentTopic)
	if err != nil {
		logutils.ZapLogger().Error("failed to decode message", zap.Error(err))
		return nil
	}

	result.PubsubTopic = watcher.PubsubTopic
	result.ContentTopic = ct

	return result
}
