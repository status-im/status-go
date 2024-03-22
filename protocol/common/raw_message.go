package common

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/protobuf"
)

type CommKeyExMsgType uint8

const (
	KeyExMsgNone  CommKeyExMsgType = 0
	KeyExMsgReuse CommKeyExMsgType = 1
	KeyExMsgRekey CommKeyExMsgType = 2
)

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

// RawMessage represent a sent or received message, kept for being able
// to re-send/propagate
type RawMessage struct {
	ID          string
	LocalChatID string
	LastSent    uint64
	SendCount   int
	Sent        bool
	// don't wrap message into ProtocolMessage.
	// when this is true, the message will not be resent via ResendTypeDataSync, but it's possible to
	// resend it via ResendTypeRawMessage specified in ResendType
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
	CommunityKeyExMsgType CommKeyExMsgType
	Ephemeral             bool
	BeforeDispatch        func(*RawMessage) error
	HashRatchetGroupID    []byte
	PubsubTopic           string
	ResendType            ResendType
	ResendMethod          ResendMethod
}
