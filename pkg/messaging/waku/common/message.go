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

// ReceivedMessage is the adapter's view of a data packet received through the
// WakuV2 protocol. It stays encrypted here: decoding and matching happen in
// the transport (status-im/status-go#7464).
type ReceivedMessage struct {
	Envelope Envelope // Wrapped Waku Message

	MsgType MessageType

	Sent uint32 // Time when the message was posted into the network in seconds

	PubsubTopic  string
	ContentTopic TopicType

	hash common.Hash
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
