package common

import (
	"crypto/ecdsa"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

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
