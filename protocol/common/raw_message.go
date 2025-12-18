package common

import (
	"crypto/ecdsa"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

// RawMessage represent a sent or received message, kept for being able
// to re-send/propagate
type RawMessage struct {
	ID          string
	LocalChatID string
	LastSent    uint64
	SendCount   int
	Sent        bool
	// SkipEncryptionLayer instructs the sender to bypass the encryption-layer protobuf wrapper.
	// Both public and private messages normally include that wrapper so the X3DH bundle travels with
	// them, though only private messages encrypt their payload.
	// Use this when the message must avoid exposing our bundle (e.g. anonymous sends or publishing with
	// a non-identity key).
	// With this flag set, MVDS resends are unavailable because they require encrypted
	// payloads, but RawMessage-based retries remain possible via ResendTypeRawMessage.
	SkipEncryptionLayer   bool
	SendPushNotification  bool
	MessageType           protobuf.ApplicationMetadataMessage_Type
	Payload               []byte
	Sender                *ecdsa.PrivateKey
	Recipients            []*ecdsa.PublicKey
	SkipGroupMessageWrap  bool
	SkipApplicationWrap   bool
	SendOnPersonalTopic   bool
	CommunityID           []byte
	CommunityKeyExMsgType messagingtypes.CommKeyExMsgType
	Ephemeral             bool
	BeforeDispatch        func(*RawMessage) error
	HashRatchetGroupID    []byte
	ContentTopic          string
	PubsubTopic           string
	ResendType            ResendType
	ResendMethod          ResendMethod
	Priority              *messagingtypes.MessagePriority
}

// ResendType There are distinct mechanisms for retrying send messages: Datasync supports only direct messages (1-to-1 or private group chats)
// because it requires an acknowledgment (ACK). As implemented, sending a message to a community, where hundreds of
// people receive it, would lead all recipients to attempt sending an ACK, resulting in an excessive number of messages.
// Datasync utilizes ACKs, but community messages do not, to avoid this issue. However, we still aim to retry sending
// community messages if they fail to send or if we are offline.
type ResendType uint8

const (
	// ResendTypeNone won't resend
	ResendTypeNone ResendType = 0
	// ResendTypeDataSync use DataSync which use MVDS as underlying dependency to resend messages.
	// Works only when underlying sending method is MessageSender#SendPrivate. Pls see SendPrivate for more details.
	// For usage example, you can find usage with this type value in this project. e.g. Messenger#syncContact
	ResendTypeDataSync ResendType = 1
	// ResendTypeRawMessage We have a function, watchExpiredMessages, that monitors the 'raw_messages' table
	// and will attempts to resend messages if a previous message sending failed.
	ResendTypeRawMessage ResendType = 2
)

// ResendMethod defines how to resend a raw message
type ResendMethod uint8

const (
	// ResendMethodDynamic determined by logic of Messenger#dispatchMessage, mostly based on chat type
	ResendMethodDynamic ResendMethod = 0
	// ResendMethodSendPrivate corresponding function MessageSender#SendPrivate
	ResendMethodSendPrivate ResendMethod = 1
	// ResendMethodSendCommunityMessage corresponding function MessageSender#SendCommunityMessage
	ResendMethodSendCommunityMessage ResendMethod = 2
)
